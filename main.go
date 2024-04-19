package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

var db *sql.DB

type User struct {
	UserID           int    `json:"user_id"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	Role             string `json:"role"`
	OtherUserDetails string `json:"other_user_details"`
}

type Notification struct {
	NotificationID           int    `json:"notification_id"`
	Content                  string `json:"content"`
	Timestamp                string `json:"timestamp"`
	OtherNotificationDetails string `json:"other_notification_details"`
}

func main() {
	var err error
	db, err = sql.Open("postgres", "postgresql://postgres:thu123@localhost/test?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := gin.Default()

	router.POST("/register", registerUser)
	router.POST("/login", loginUser)
	router.GET("/notifications/all", getAllNotifications)
	router.GET("/notifications/user/:user_id", getUserNotifications)

	router.Run(":8080")
}

func registerUser(c *gin.Context) {
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	var existingUser User
	err := db.QueryRow("select user_id, username, password, role, other_user_details from \"User\" where username = $1", user.Username).Scan(&existingUser.UserID, &existingUser.Username, &existingUser.Password, &existingUser.Role, &existingUser.OtherUserDetails)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
		return
	} else if err != sql.ErrNoRows {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	passwordHash := md5.Sum([]byte(user.Password))
	user.Password = hex.EncodeToString(passwordHash[:])

	_, err = db.Exec("insert into \"User\" (user_id, username, password, role, other_user_details) values ($1, $2, $3, $4, $5)",
		user.UserID, user.Username, user.Password, user.Role, user.OtherUserDetails)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

func loginUser(c *gin.Context) {
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	if user.Username == "" || user.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}
	passwordHash := md5.Sum([]byte(user.Password))
	user.Password = hex.EncodeToString(passwordHash[:])

	row := db.QueryRow("select user_id, username, role, other_user_details from \"User\" where username = $1 and password = $2",
		user.Username, user.Password)

	if err := row.Scan(&user.UserID, &user.Username, &user.Role, &user.OtherUserDetails); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login"})
		return
	}
	user.Password = ""
	c.JSON(http.StatusOK, user)
}

func getAllNotifications(c *gin.Context) {
	rows, err := db.Query("select notification_id, content, timestamp, other_notification_details from Notification")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var notification Notification
		err := rows.Scan(&notification.NotificationID, &notification.Content, &notification.Timestamp, &notification.OtherNotificationDetails)
		if err != nil {
			log.Println(err)
			continue
		}
		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

func getUserNotifications(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	rows, err := db.Query("select Notification.notification_id, content, timestamp, other_notification_details from Notification join Notification_Recipients on Notification.notification_id = Notification_Recipients.notification_id where Notification_Recipients.recipient_id = $1", userID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var notification Notification
		err := rows.Scan(&notification.NotificationID, &notification.Content, &notification.Timestamp, &notification.OtherNotificationDetails)
		if err != nil {
			log.Println(err)
			continue
		}
		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, notifications)
}
