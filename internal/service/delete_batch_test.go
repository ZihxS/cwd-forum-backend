package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gin-quickstart/internal/model"
	"gin-quickstart/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type countingLogger struct {
	gormlogger.Interface
	count int64
}

func (l *countingLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (l *countingLogger) Info(context.Context, string, ...interface{})  {}
func (l *countingLogger) Warn(context.Context, string, ...interface{})  {}
func (l *countingLogger) Error(context.Context, string, ...interface{}) {}

func (l *countingLogger) Trace(_ context.Context, _ time.Time, fc func() (string, int64), err error) {
	_, _ = fc()
	if err == nil {
		atomic.AddInt64(&l.count, 1)
	}
}

func (l *countingLogger) Reset() {
	atomic.StoreInt64(&l.count, 0)
}

func (l *countingLogger) Count() int {
	return int(atomic.LoadInt64(&l.count))
}

func newTestDB(t *testing.T) (*gorm.DB, *countingLogger) {
	t.Helper()

	logger := &countingLogger{}
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.Category{},
		&model.Thread{},
		&model.Post{},
		&model.Vote{},
		&model.Reaction{},
		&model.Tag{},
		&model.Notification{},
		&model.Attachment{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return db, logger
}

func newFakeRedis(t *testing.T) (*redis.Client, func()) {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Protocol: 2,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			server, client := net.Pipe()
			go handleFakeRedisConn(server)
			return client, nil
		},
	})

	cleanup := func() {
		_ = client.Close()
	}

	return client, cleanup
}

func handleFakeRedisConn(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	writeOK := func() {
		_, _ = writer.WriteString("+OK\r\n")
		_ = writer.Flush()
	}

	writeInt := func(n int) {
		_, _ = writer.WriteString(fmt.Sprintf(":%d\r\n", n))
		_ = writer.Flush()
	}

	writePong := func() {
		_, _ = writer.WriteString("+PONG\r\n")
		_ = writer.Flush()
	}

	writeHello := func() {
		_, _ = writer.WriteString("%7\r\n")
		_, _ = writer.WriteString("$6\r\nserver\r\n")
		_, _ = writer.WriteString("$5\r\nredis\r\n")
		_, _ = writer.WriteString("$7\r\nversion\r\n")
		_, _ = writer.WriteString("$5\r\n7.2.0\r\n")
		_, _ = writer.WriteString("$5\r\nproto\r\n")
		_, _ = writer.WriteString(":3\r\n")
		_, _ = writer.WriteString("$2\r\nid\r\n")
		_, _ = writer.WriteString(":1\r\n")
		_, _ = writer.WriteString("$4\r\nmode\r\n")
		_, _ = writer.WriteString("$10\r\nstandalone\r\n")
		_, _ = writer.WriteString("$4\r\nrole\r\n")
		_, _ = writer.WriteString("$6\r\nmaster\r\n")
		_, _ = writer.WriteString("$7\r\nmodules\r\n")
		_, _ = writer.WriteString("*0\r\n")
		_ = writer.Flush()
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		if !strings.HasPrefix(line, "*") {
			return
		}

		argc, err := strconv.Atoi(strings.TrimSpace(line[1:]))
		if err != nil {
			return
		}

		args := make([]string, 0, argc)
		for i := 0; i < argc; i++ {
			lenLine, err := reader.ReadString('\n')
			if err != nil {
				return
			}

			if !strings.HasPrefix(lenLine, "$") {
				return
			}

			argLen, err := strconv.Atoi(strings.TrimSpace(lenLine[1:]))
			if err != nil {
				return
			}

			buf := make([]byte, argLen+2)
			if _, err := io.ReadFull(reader, buf); err != nil {
				return
			}

			args = append(args, string(buf[:argLen]))
		}

		switch strings.ToUpper(args[0]) {
		case "PING":
			writePong()
		case "HELLO":
			writeHello()
		case "DEL":
			writeInt(1)
		case "CLIENT", "AUTH", "SELECT", "SET", "GET", "QUIT":
			writeOK()
		default:
			writeOK()
		}
	}
}

func setupBaseData(t *testing.T, db *gorm.DB) (model.User, model.Category) {
	t.Helper()

	user := model.User{
		Name:     "Tester",
		Username: "tester",
		Email:    "tester@example.com",
		Password: "password123",
		Role:     "user",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	category := model.Category{
		Name:        "General",
		Slug:        "general",
		Description: "General",
	}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	return user, category
}

func TestThreadService_Create_BatchesTagLookup(t *testing.T) {
	db, counter := newTestDB(t)
	redisClient, cleanup := newFakeRedis(t)
	defer cleanup()

	user, category := setupBaseData(t, db)
	for i := 0; i < 5; i++ {
		tag := model.Tag{Name: fmt.Sprintf("Tag %d", i), Slug: fmt.Sprintf("tag-%d", i)}
		if err := db.Create(&tag).Error; err != nil {
			t.Fatalf("create tag: %v", err)
		}
	}

	var tags []model.Tag
	if err := db.Find(&tags).Error; err != nil {
		t.Fatalf("load tags: %v", err)
	}

	tagIDs := make([]uint, 0, len(tags))
	for _, tag := range tags {
		tagIDs = append(tagIDs, tag.ID)
	}

	repo := repository.NewThreadRepository(db, redisClient)
	svc := NewThreadService(repo)

	counter.Reset()
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	thread, post, err := svc.Create(
		category.ID,
		"Thread title",
		"thread-title",
		"Thread body",
		user.ID,
		tagIDs,
		ctx,
	)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	if thread == nil || post == nil {
		t.Fatalf("expected thread and post")
	}

	if got := counter.Count(); got > 8 {
		t.Fatalf("too many sql queries for batch tag create: %d", got)
	}

	var loaded model.Thread
	if err := db.Preload("Tags").First(&loaded, thread.ID).Error; err != nil {
		t.Fatalf("load thread: %v", err)
	}

	if len(loaded.Tags) != len(tagIDs) {
		t.Fatalf("expected %d tags, got %d", len(tagIDs), len(loaded.Tags))
	}
}

func TestThreadService_Delete_BulkDeletesLargeThread(t *testing.T) {
	db, counter := newTestDB(t)
	redisClient, cleanup := newFakeRedis(t)
	defer cleanup()

	user, category := setupBaseData(t, db)
	thread := model.Thread{
		CategoryID: category.ID,
		AuthorID:   user.ID,
		Title:      "Thread",
		Slug:       "thread",
	}
	if err := db.Create(&thread).Error; err != nil {
		t.Fatalf("create thread: %v", err)
	}

	root := model.Post{
		ThreadID: thread.ID,
		AuthorID: user.ID,
		Content:  "root",
	}
	if err := db.Create(&root).Error; err != nil {
		t.Fatalf("create root post: %v", err)
	}

	for i := 0; i < 40; i++ {
		reply := model.Post{
			ThreadID: thread.ID,
			AuthorID: user.ID,
			ParentID: &root.ID,
			Content:  fmt.Sprintf("reply %d", i),
		}
		if err := db.Create(&reply).Error; err != nil {
			t.Fatalf("create reply: %v", err)
		}

		if i%10 == 0 {
			grandChild := model.Post{
				ThreadID: thread.ID,
				AuthorID: user.ID,
				ParentID: &reply.ID,
				Content:  fmt.Sprintf("grandchild %d", i),
			}
			if err := db.Create(&grandChild).Error; err != nil {
				t.Fatalf("create grandchild: %v", err)
			}
			if err := db.Create(&model.Attachment{PostID: grandChild.ID, UploaderId: user.ID, Filename: "a", MimeType: "text/plain", FileSize: 1, Url: "u"}).Error; err != nil {
				t.Fatalf("create attachment: %v", err)
			}
		}

		if err := db.Create(&model.Vote{PostID: reply.ID, UserID: user.ID, Value: 1}).Error; err != nil {
			t.Fatalf("create vote: %v", err)
		}
		if err := db.Create(&model.Reaction{PostId: reply.ID, UserId: user.ID, Emoji: "like"}).Error; err != nil {
			t.Fatalf("create reaction: %v", err)
		}
	}

	counter.Reset()
	svc := NewThreadService(repository.NewThreadRepository(db, redisClient))
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	if err := svc.Delete(uint64(thread.ID), ctx); err != nil {
		t.Fatalf("delete thread: %v", err)
	}

	if got := counter.Count(); got > 10 {
		t.Fatalf("too many sql queries for bulk thread delete: %d", got)
	}

	var threadsCount, postsCount, votesCount, reactionsCount, attachmentsCount, threadTagsCount, notificationsCount int64
	if err := db.Model(&model.Thread{}).Count(&threadsCount).Error; err != nil {
		t.Fatalf("count threads: %v", err)
	}
	if err := db.Model(&model.Post{}).Count(&postsCount).Error; err != nil {
		t.Fatalf("count posts: %v", err)
	}
	if err := db.Model(&model.Vote{}).Count(&votesCount).Error; err != nil {
		t.Fatalf("count votes: %v", err)
	}
	if err := db.Model(&model.Reaction{}).Count(&reactionsCount).Error; err != nil {
		t.Fatalf("count reactions: %v", err)
	}
	if err := db.Model(&model.Attachment{}).Count(&attachmentsCount).Error; err != nil {
		t.Fatalf("count attachments: %v", err)
	}
	if err := db.Table("thread_tags").Count(&threadTagsCount).Error; err != nil {
		t.Fatalf("count thread_tags: %v", err)
	}
	if err := db.Model(&model.Notification{}).Count(&notificationsCount).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}

	if threadsCount != 0 || postsCount != 0 || votesCount != 0 || reactionsCount != 0 || attachmentsCount != 0 || threadTagsCount != 0 || notificationsCount != 0 {
		t.Fatalf("unexpected remaining rows: threads=%d posts=%d votes=%d reactions=%d attachments=%d thread_tags=%d notifications=%d",
			threadsCount, postsCount, votesCount, reactionsCount, attachmentsCount, threadTagsCount, notificationsCount)
	}
}

func TestPostService_Delete_BulkDeletesLargeReplyTree(t *testing.T) {
	db, counter := newTestDB(t)
	redisClient, cleanup := newFakeRedis(t)
	defer cleanup()

	user, category := setupBaseData(t, db)
	thread := model.Thread{
		CategoryID: category.ID,
		AuthorID:   user.ID,
		Title:      "Thread",
		Slug:       "thread",
	}
	if err := db.Create(&thread).Error; err != nil {
		t.Fatalf("create thread: %v", err)
	}

	root := model.Post{
		ThreadID: thread.ID,
		AuthorID: user.ID,
		Content:  "root",
	}
	if err := db.Create(&root).Error; err != nil {
		t.Fatalf("create root: %v", err)
	}

	var lastLevel []model.Post
	for i := 0; i < 20; i++ {
		reply := model.Post{
			ThreadID: thread.ID,
			AuthorID: user.ID,
			ParentID: &root.ID,
			Content:  fmt.Sprintf("reply %d", i),
		}
		if err := db.Create(&reply).Error; err != nil {
			t.Fatalf("create reply: %v", err)
		}
		lastLevel = append(lastLevel, reply)
	}

	for _, parent := range lastLevel[:10] {
		for i := 0; i < 2; i++ {
			child := model.Post{
				ThreadID: thread.ID,
				AuthorID: user.ID,
				ParentID: &parent.ID,
				Content:  fmt.Sprintf("child %d", i),
			}
			if err := db.Create(&child).Error; err != nil {
				t.Fatalf("create child: %v", err)
			}

			if err := db.Create(&model.Attachment{PostID: child.ID, UploaderId: user.ID, Filename: "a", MimeType: "text/plain", FileSize: 1, Url: "u"}).Error; err != nil {
				t.Fatalf("create attachment: %v", err)
			}
			if err := db.Create(&model.Vote{PostID: child.ID, UserID: user.ID, Value: 1}).Error; err != nil {
				t.Fatalf("create vote: %v", err)
			}
			if err := db.Create(&model.Reaction{PostId: child.ID, UserId: user.ID, Emoji: "like"}).Error; err != nil {
				t.Fatalf("create reaction: %v", err)
			}
			if err := db.Create(&model.Notification{PostId: &child.ID, UserId: user.ID, Type: "reply", Payload: "{}"}).Error; err != nil {
				t.Fatalf("create notification: %v", err)
			}
		}
	}

	counter.Reset()
	svc := NewPostService(repository.NewPostRepository(db, redisClient))
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	if err := svc.Delete(uint64(root.ID), ctx); err != nil {
		t.Fatalf("delete post tree: %v", err)
	}

	if got := counter.Count(); got > 10 {
		t.Fatalf("too many sql queries for bulk post delete: %d", got)
	}

	var remaining int64
	for _, table := range []string{"posts", "votes", "reactions", "attachments", "notifications"} {
		if err := db.Table(table).Count(&remaining).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if remaining != 0 {
			t.Fatalf("expected table %s empty, got %d", table, remaining)
		}
	}
}
