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
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Password string `json:"password"`
	RoleID   int    `json:"role_id"`
	// Role             string `json:"role"`
	// OtherUserDetails string `json:"other_user_details"`
}

type Notification struct {
	NotificationID int    `json:"notification_id"`
	Title          string `json: "title"`
	Content        string `json:"content"`
	Timestamp      string `json:"timestamp"`
	// OtherNotificationDetails string `json:"other_notification_details"`
}

func main() {
	var err error
	db, err = sql.Open("postgres", "postgresql://postgres:thu123@localhost/test2?sslmode=disable")
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
	err := db.QueryRow("select username from users where username = $1", user.Username).Scan(&existingUser.UserID, &existingUser.Username, &existingUser.Password, &existingUser.RoleID)
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

	_, err = db.Exec("insert into users (username, password, role_id) values ($1, $2, $3)",
		user.Username, user.Password, user.RoleID)
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

	row := db.QueryRow("select user_id, username, role_id from users where username = $1 and password = $2",
		user.Username, user.Password)

	if err := row.Scan(&user.UserID, &user.Username, &user.RoleID); err != nil {
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
	rows, err := db.Query("select notification_id, title, message, created_at from notifications")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var notification Notification
		err := rows.Scan(&notification.NotificationID, &notification.Title, &notification.Content, &notification.Timestamp)
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

	rows, err := db.Query(` select n.notification_id, n.title, n.message, n.created_at
	from notifications n
	inner join notification_users nu on n.notification_id = nu.notification_id
	where nu.user_id = $1
	
	union
	
	select n.notification_id, n.title, n.message, n.created_at
	from notifications n
	inner join notification_roles nr on n.notification_id = nr.notification_id
	where nr.role_id in (select role_id from users where user_id = $1)
	
	order by notification_id asc;
	`, userID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var notification Notification
		err := rows.Scan(&notification.NotificationID, &notification.Title, &notification.Content, &notification.Timestamp)
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
