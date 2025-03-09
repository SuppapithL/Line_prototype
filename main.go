package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	_ "github.com/lib/pq"
	"github.com/line/line-bot-sdk-go/linebot"
)

var (
	bot      *linebot.Client
	db       *sql.DB
	userFile = make(map[string]string) // Temporary session for filename per user
	mu       sync.Mutex                // Ensures safe concurrent access
	s3Client *s3.Client
	bucket   string
)

func main() {
	var err error

	// Initialize LINE Bot Client
	channelSecret := "5ad6c664cb50426b820adc87f1be70c7"
	channelToken := "1OCW+4wHULcJbXPT1hnRCsNs4lH34qPhBMkr/0w6hP0L98aqKV8j2aSFuyJdKc9LC7kQ9zWBHW0NBtQ1GyVuXOdGrH0vUbemhee6Xdc37WONzASa1Q/couQA0v9n0/kz1n5fSpa8ukGK45ybV0wj2QdB04t89/1O/w1cDnyilFU="
	bot, err = linebot.New(channelSecret, channelToken)
	if err != nil {
		log.Fatalf("Error creating LINE bot client: %v", err)
	}

	// Connect to PostgreSQL
	dbConnStr := "postgres://postgres:Suppapith2@localhost:5432/file_manager?sslmode=disable"
	db, err = sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Error connecting to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Initialize R2 (AWS S3-compatible)
	s3Client, bucket, err = initR2()
	if err != nil {
		log.Fatalf("Error initializing R2: %v", err)
	}

	// Set up HTTP server
	http.HandleFunc("/callback", callbackHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server is running at port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// callbackHandler processes incoming webhook events
func callbackHandler(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				handleTextMessage(event, message)
			case *linebot.ImageMessage, *linebot.FileMessage:
				userID := event.Source.UserID
				mu.Lock()
				_, exists := userFile[userID] // Check if user started an upload
				mu.Unlock()

				if exists {
					handleFileMessage(event, message) // ✅ Process file if upload was started
				} else {
					bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Please use 'upload -category(optional) -filename' first before sending a file.")).Do()
				}
			default:
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Use 'upload' to upload\nUse 'open' to open files")).Do()
			}
		}
	}

}

// handleTextMessage processes text commands
func handleTextMessage(event *linebot.Event, message linebot.Message) {
	userID := event.Source.UserID

	mu.Lock()
	filename, exists := userFile[userID]
	mu.Unlock()

	// ✅ First, check if it's a text message
	if textMessage, ok := message.(*linebot.TextMessage); ok {
		// Process text message
		command := strings.Fields(textMessage.Text) // ✅ Now it's safe to access .Text
		if len(command) == 0 {
			return
		}

		switch command[0] {
		case "upload":
			if len(command) < 2 {
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Usage: upload [category] filename")).Do()
				return
			}

			category := "default"
			filename := ""

			if len(command) == 2 {
				filename = command[1]
			} else {
				category = command[1]
				filename = command[2]
			}

			if filename == "" {
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Error: filename cannot be empty")).Do()
				return
			}

			timestamp := time.Now().Format("2006-01-02 15:04:05")

			if err := insertFileMetadata(userID, filename, category, timestamp); err != nil {
				log.Printf("Error inserting metadata: %v", err)
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Error saving file metadata.")).Do()
				return
			}

			mu.Lock()
			userFile[userID] = filename
			mu.Unlock()

			bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Send file:")).Do()

		case "open":
			if len(command) < 2 {
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Usage: open filename")).Do()
				return
			}
			filesad := command[1]

			// 🔥 Get the actual filename from R2 (ignoring extension issues)
			fileURL, err := getFileURL(filesad)
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				filename := filepath.Base(fileURL) // Extract filename from URL
				fmt.Println("File URL:", fileURL)
				fmt.Println("Extracted filename:", filename)
			}
			if err != nil {
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Error: File not found in R2.")).Do()
				return
			}
			filename = strings.TrimSpace(filepath.Base(fileURL))

			if filename == "" {
				fmt.Println("Error: Filename extraction failed.")
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Error: Could not determine file name.")).Do()
				return
			}
			fmt.Println("Extracted filename:", filename)

			fmt.Printf("Filename raw: [%s]\n", filename)
			// 🔥 Improved file type detection based on the actual filename
			switch {
			case strings.HasSuffix(filename, ".txt"):
				fmt.Println("Detected .txt file") // Debugging
				fmt.Println("Fetching text from URL:", fileURL)
				content, err := fetchTextFromURL(fileURL)
				if err != nil {
					fmt.Println("Error fetching file content:", err) // Debugging
					bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Error reading file content.")).Do()
					return
				}
				fmt.Println("File content:", content) // Debugging
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(content)).Do()

			case strings.HasSuffix(filename, ".jpeg"), strings.HasSuffix(filename, ".jpg"), strings.HasSuffix(filename, ".png"):
				fmt.Println("Detected image file") // Debugging
				bot.ReplyMessage(event.ReplyToken, linebot.NewImageMessage(fileURL, fileURL)).Do()

			default:
				fmt.Println("Unsupported file type") // Debugging
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Unsupported file type.")).Do()
			}
		case "list":
			var category string
			if len(command) < 2 {
				category = "" // No category specified, list all available categories
			} else {
				category = command[1]
			}

			files, err := listFilesFromDB(category) // Function to fetch files from PostgreSQL
			if err != nil {
				log.Println("Database query error:", err)
				return
			}

			if len(files) == 0 {
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("No files found.")).Do()
				return
			}

			// Format the file list
			fileListMsg := "Files:\n" + strings.Join(files, "\n")
			bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(fileListMsg)).Do()

		case "rename":
			if len(command) < 3 {
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Usage: rename <old_filename> <new_filename>")).Do()
				return
			}

			oldFilename := command[1]
			newFilename := command[2]

			err := renameFileInDB(oldFilename, newFilename)
			if err != nil {
				log.Println("Rename error:", err)
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Error renaming file.")).Do()
				return
			}
			bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("File renamed successfully!")).Do()
			return
		case "delete":
			if len(command) < 2 {
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Usage: delete <filename>")).Do()
				return
			}

			filename := command[1]
			fileURL, err := getFileURL(filename)
			if err != nil {
				fmt.Println("Error:", err)
			}
			// Call function to delete file from R2 & Database
			err = deleteFile(fileURL, filename)
			if err != nil {
				log.Println("Delete error:", err)
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Error deleting file.")).Do()
				return
			}

			bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("File deleted successfully!")).Do()

		default:
			if exists {
				// ✅ If a filename is set, handle the text as a file upload
				fileData := []byte(textMessage.Text)
				txtFilename := filename + ".txt"
				fileURL, err := uploadToR2(txtFilename, fileData)
				if err != nil {
					log.Printf("Error uploading text file to R2: %v", err)
					return
				}
				updateFileURL(filename, fileURL)
				mu.Lock()
				delete(userFile, userID)
				mu.Unlock()
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Upload successful!")).Do()
			} else {
				bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("USAGE:\nupload,open,list,rename")).Do()
			}
		}
		return // ✅ Return after processing text message
	}

	// ✅ Move file handling inside `if exists`
	if exists {
		switch msg := message.(type) {
		case *linebot.ImageMessage, *linebot.FileMessage:
			// ✅ Call handleFileMessage to process images/files
			handleFileMessage(event, msg)
		default:
			// ❌ Reject unsupported messages
			bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Unsupported message type. Please send text, image, or file.")).Do()
		}
	} else {
		bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Please use 'upload -category(optional) -filename' first.")).Do()
	}
}

func handleFileMessage(event *linebot.Event, message linebot.Message) {
	userID := event.Source.UserID

	mu.Lock()
	filename, exists := userFile[userID]
	mu.Unlock()

	if !exists {
		bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Please use 'upload category filename' first.")).Do()
		return
	}

	var fileData []byte
	var ext string

	switch msg := message.(type) {
	case *linebot.FileMessage:
		// ✅ จัดการกับไฟล์ที่แนบมา
		log.Printf("Received file message: %s", msg.FileName)
		content, err := bot.GetMessageContent(msg.ID).Do()
		if err != nil {
			log.Printf("Error getting file content: %v", err)
			bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Error retrieving file.")).Do()
			return
		}
		defer content.Content.Close()

		fileData, err = io.ReadAll(content.Content)
		if err != nil {
			log.Printf("Error reading file content: %v", err)
			bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Error reading file.")).Do()
			return
		}
		log.Printf("File size: %d bytes", len(fileData))

		// 🔥 ตรวจสอบไฟล์โดยใช้ Content-Type
		contentType := http.DetectContentType(fileData)
		switch contentType {
		case "image/png":
			ext = ".png"
		case "image/jpeg":
			ext = ".jpeg"
		default:
			ext = filepath.Ext(msg.FileName) // ใช้ extension เดิมถ้ารู้จัก
		}

	case *linebot.ImageMessage:
		// ✅ จัดการกับรูปภาพที่แนบมา
		log.Printf("Received image message")
		content, err := bot.GetMessageContent(msg.ID).Do()
		if err != nil {
			log.Printf("Error getting image content: %v", err)
			bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Error retrieving image.")).Do()
			return
		}
		defer content.Content.Close()

		fileData, err = io.ReadAll(content.Content)
		if err != nil {
			log.Printf("Error reading image content: %v", err)
			bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Error reading image.")).Do()
			return
		}
		ext = ".jpeg" // LINE ส่งภาพมาเป็น JPEG เสมอ

	default:
		bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Unsupported file type. Only images and files are allowed.")).Do()
		return
	}

	fullFilename := filename + ext
	log.Printf("Uploading file: %s", fullFilename)

	fileURL, err := uploadToR2(fullFilename, fileData)
	if err != nil {
		log.Printf("Error uploading file to R2: %v", err)
		bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Error uploading file.")).Do()
		return
	}

	log.Printf("Uploaded file URL: %s", fileURL)
	updateFileURL(filename, fileURL)

	// ✅ อัปเดตและล้างข้อมูลผู้ใช้หลังจากอัปโหลดเสร็จ
	mu.Lock()
	delete(userFile, userID)
	mu.Unlock()

	bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("Upload successful!")).Do()
}

func insertFileMetadata(userID, filename, theme, url string) error {
	query := `INSERT INTO line_01 (user_id, file_name, file_content, created_at, theme) 
              VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(query, userID, filename, url, time.Now().Format("2006-01-02 15:04:05"), theme)
	return err
}

func updateFileURL(filename, url string) error {
	query := `UPDATE line_01 SET file_content = $1 WHERE file_name = $2`
	_, err := db.Exec(query, url, filename)
	return err
}

func initR2() (*s3.Client, string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"3ba7d5427f34a754ca877390945b67e5",
			"2c202e05c96a3b0603168a4d6e0446389e29b67ae61814ae36c2abc9a4bda40c",
			"",
		)),
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load R2 configuration: %w", err)
	}

	// ✅ Use BaseEndpoint (New Method)
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("https://62aec46dee2be140c039696df2e6a205.r2.cloudflarestorage.com")
		o.UsePathStyle = true // Required for R2
	})

	bucketName := "line-bot-datababa"
	return s3Client, bucketName, nil
}

func uploadToR2(filename string, data []byte) (string, error) {
	// Auto-detect file content type
	contentType := http.DetectContentType(data)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType), // Important for proper file handling
	}

	_, err := s3Client.PutObject(context.TODO(), input)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to R2: %v", err)
	}

	bucketID := "pub-5100b97c44f44bd0b69047096448e186" // Replace with your actual R2.dev bucket ID
	return fmt.Sprintf("https://%s.r2.dev/%s", bucketID, filename), nil
}

func fetchTextFromURL(fileURL string) (string, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}

	resp, err := client.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("error fetching file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: received status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading file content: %w", err)
	}

	return string(body), nil
}

func getFileURL(filename string) (string, error) {
	var fileURL string

	// 🔹 Check the database first (only get file_content, no file_name)
	err := db.QueryRow("SELECT file_content FROM line_01 WHERE file_name = $1", filename).Scan(&fileURL)
	if err == nil {
		// If the file URL is found in the DB, return it
		return fileURL, nil
	}

	// 🔹 If not found in DB, check R2
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	result, err := s3Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		return "", fmt.Errorf("error listing R2 objects: %w", err)
	}

	var correctFile string
	for _, obj := range result.Contents {
		if strings.HasPrefix(*obj.Key, filename) { // Match filename ignoring extension
			correctFile = *obj.Key
			break
		}
	}

	if correctFile == "" {
		return "", fmt.Errorf("file not found in R2")
	}

	// 🔹 Construct the file URL
	fileURL = fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s", bucket, correctFile)

	// 🔹 Update the database with the correct URL (but NOT the real filename)
	_, err = db.Exec("UPDATE line_01 SET file_content = $1 WHERE file_name = $2", fileURL, filename)
	if err != nil {
		log.Printf("Warning: Could not update DB with R2 file URL: %v", err)
	}

	// 🔹 Extract filename from URL
	extractedFilename := filepath.Base(fileURL)
	fmt.Println("File URL:", fileURL)
	fmt.Println("Extracted filename:", extractedFilename)

	return fileURL, nil
}

func listR2Objects(s3Client *s3.Client, bucket string) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	result, err := s3Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		log.Fatalf("Error listing R2 objects: %v", err)
	}

	for _, obj := range result.Contents {
		log.Printf("File: %s, Size: %d bytes", *obj.Key, obj.Size)
	}
}
func listFilesFromDB(category string) ([]string, error) {
	var rows *sql.Rows
	var err error

	if category == "" {
		rows, err = db.Query("SELECT DISTINCT theme FROM line_01") // List all categories
	} else {
		rows, err = db.Query("SELECT file_name FROM line_01 WHERE theme = $1", category) // List files in the category
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result = append(result, name)
	}

	return result, nil
}
func renameFileInDB(oldFilename, newFilename string) error {
	_, err := db.Exec("UPDATE line_01 SET file_name = $1 WHERE file_name = $2", newFilename, oldFilename)
	return err
}
func deleteFile(fileURL, filename string) error {
	s3Client, bucketName, err := initR2() // ✅ Initialize R2 client
	if err != nil {
		return fmt.Errorf("failed to initialize R2: %w", err)
	}

	// 🔹 Extract filename from URL
	fileKey := filepath.Base(fileURL) // Ensure consistent naming

	// 🗑️ Delete from R2
	_, err = s3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: &bucketName,
		Key:    &fileKey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}

	// 🗑️ Delete from Database
	_, err = db.Exec("DELETE FROM line_01 WHERE file_name = $1", filename)
	if err != nil {
		return fmt.Errorf("failed to delete from DB: %w", err)
	}

	return nil // ✅ Success
}
