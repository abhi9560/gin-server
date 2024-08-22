package main

import (
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Alarm represents an alarm with a specific time and message
type Alarm struct {
	Time      time.Time
	Message   string
	Completed bool // Track if the alarm has been completed
}

// Global slice to store all alarms and a mutex to handle concurrent access
var (
	alarms        []Alarm
	activeAlarms  = make(map[string]*time.Timer)
	mu            sync.Mutex
)

// SetAlarm sets an alarm for a specific time with a message
func SetAlarm(alarmTime time.Time, message string) {
	duration := alarmTime.Sub(time.Now())
	if duration <= 0 {
		fmt.Println("The alarm time is in the past! Please set a future time.")
		return
	}

	fmt.Printf("Alarm set for %s\n", alarmTime.Format(time.RFC1123))

	mu.Lock()
	defer mu.Unlock()

	alarms = append(alarms, Alarm{Time: alarmTime, Message: message})

	// Create a new timer and store it
	timer := time.AfterFunc(duration, func() {
		TriggerAlarm(message)
	})
	activeAlarms[message] = timer

	fmt.Printf("Alarm set for %s\n", alarmTime.Format(time.RFC1123))
}

// TriggerAlarm handles the alarm when it goes off
func TriggerAlarm(message string) {
	fmt.Printf("\nALARM! %s\n", message)

	mu.Lock()
	defer mu.Unlock()

	for i, alarm := range alarms {
		if alarm.Message == message {
			alarms[i].Completed = true
			break
		}
	}

	delete(activeAlarms, message)
}

// CountTodaysAlarms returns the number of alarms set for today
func CountTodaysAlarms() int {
	mu.Lock()
	defer mu.Unlock()

	count := 0
	today := time.Now().Format("2006-01-02")
	for _, alarm := range alarms {
		if alarm.Time.Format("2006-01-02") == today {
			count++
		}
	}
	return count
}

// RescheduleAlarm allows rescheduling an existing alarm
func RescheduleAlarm(message string, newTime time.Time) bool {
	mu.Lock()
	defer mu.Unlock()

	for i, alarm := range alarms {
		if alarm.Message == message {
			// Cancel the existing alarm
			if timer, ok := activeAlarms[message]; ok {
				timer.Stop()
				delete(activeAlarms, message)
			}

			// Update the alarm time
			alarms[i].Time = newTime
			alarms[i].Completed = false // Reset completed status

			// Set the new alarm
			SetAlarm(newTime, message)
			return true
		}
	}
	return false
}

// StopAlarm stops an active alarm
func StopAlarm(message string) bool {
	mu.Lock()
	defer mu.Unlock()

	for i, alarm := range alarms {
		if alarm.Message == message {
			if timer, ok := activeAlarms[message]; ok {
				timer.Stop()
				delete(activeAlarms, message)
			}
			alarms[i].Completed = true
			return true
		}
	}
	return false
}

// GetCompletedAlarms returns a list of completed alarms
func GetCompletedAlarms() []Alarm {
	mu.Lock()
	defer mu.Unlock()

	completedAlarms := []Alarm{}
	for _, alarm := range alarms {
		if alarm.Completed {
			completedAlarms = append(completedAlarms, alarm)
		}
	}
	return completedAlarms
}

// GetAlarmsForDisplay returns a copy of all alarms for display purposes
func GetAlarmsForDisplay() []Alarm {
	mu.Lock()
	defer mu.Unlock()

	return append([]Alarm{}, alarms...)
}

func main() {
	// Initialize Gin router
	router := gin.Default()

	// Load HTML templates
	router.SetHTMLTemplate(template.Must(template.ParseFiles(
		"templates/index.html", 
		"templates/alarm.html", 
		"templates/completed_alarms.html")))

	// Serve the form to the user and show the number of alarms set for today
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"TodayAlarmCount": CountTodaysAlarms(),
			"Alarms":          GetAlarmsForDisplay(),
		})
	})

	// Handle alarm display
	router.GET("/alarm/:message", func(c *gin.Context) {
		message := c.Param("message")
		c.HTML(http.StatusOK, "alarm.html", gin.H{
			"Message": message,
		})
	})

	// Handle alarm checking
	router.GET("/check_alarm", func(c *gin.Context) {
		mu.Lock()
		defer mu.Unlock()

		for _, alarm := range alarms {
			if !alarm.Completed && alarm.Time.Before(time.Now()) {
				c.JSON(http.StatusOK, gin.H{"message": alarm.Message})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": ""}) // No alarm to trigger
	})

	// Handle form submission
	router.POST("/set_alarm", func(c *gin.Context) {
		alarmTimeStr := c.PostForm("alarm_time")
		message := c.PostForm("message")

		alarmTime, err := time.Parse("2006-01-02T15:04", alarmTimeStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid time format. Please use the correct format.")
			return
		}

		SetAlarm(alarmTime, message)
		c.Redirect(http.StatusFound, "/")
	})

	// Handle rescheduling alarms
	router.POST("/reschedule_alarm", func(c *gin.Context) {
		message := c.PostForm("message")
		newTimeStr := c.PostForm("new_time")

		newTime, err := time.Parse("2006-01-02T15:04", newTimeStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid time format. Please use the correct format.")
			return
		}

		if RescheduleAlarm(message, newTime) {
			c.Redirect(http.StatusFound, "/")
		} else {
			c.String(http.StatusNotFound, "Alarm not found.")
		}
	})

	// Handle stopping alarms
	router.POST("/stop_alarm", func(c *gin.Context) {
		message := c.PostForm("message")

		if StopAlarm(message) {
			c.Redirect(http.StatusFound, "/")
		} else {
			c.String(http.StatusNotFound, "Alarm not found.")
		}
	})

	// Display completed alarms
	router.GET("/completed_alarms", func(c *gin.Context) {
		c.HTML(http.StatusOK, "completed_alarms.html", gin.H{
			"CompletedAlarms": GetCompletedAlarms(),
		})
	})

	// Start the server
	router.Run(":8080")
}
