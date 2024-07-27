// tasks/email.go
package tasks

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
	"gopkg.in/gomail.v2"
)

var RedisClient *redis.Client

func init() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})
}

type EmailTask struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

func QueueVerificationEmail(email, otp string) error {
	task := EmailTask{
		Email: email,
		OTP:   otp,
	}

	jsonTask, err := json.Marshal(task)
	if err != nil {
		return err
	}

	err = RedisClient.RPush(RedisClient.Context(), "email_queue", jsonTask).Err()
	if err != nil {
		return err
	}

	log.Printf("Verification email for %s queued", email)
	return nil
}

func ProcessEmailQueue() {
	for {
		result, err := RedisClient.BLPop(RedisClient.Context(), 0, "email_queue").Result()
		if err != nil {
			log.Printf("Error popping from queue: %v", err)
			continue
		}

		var task EmailTask
		err = json.Unmarshal([]byte(result[1]), &task)
		if err != nil {
			log.Printf("Error unmarshaling task: %v", err)
			continue
		}

		err = sendEmail(task.Email, task.OTP)
		if err != nil {
			log.Printf("Error sending email: %v", err)
			// Optionally, you could re-queue the task or implement a retry mechanism
		}
	}
}

func sendEmail(email, otp string) error {
	// Load email template
	tmpl, err := template.ParseFiles("templates/email_template.html")
	if err != nil {
		return err
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, struct{ OTP string }{OTP: otp}); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", "Restaurant System <"+os.Getenv("SMTP_EMAIL")+">")
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Account Verification")
	m.SetBody("text/html", body.String())

	smtpPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		return err
	}

	d := gomail.NewDialer(
		os.Getenv("SMTP_HOST"),
		smtpPort,
		os.Getenv("SMTP_EMAIL"),
		os.Getenv("SMTP_PASSWORD"),
	)

	return d.DialAndSend(m)
}

func QueueResetPasswordEmail(email, otp string) error {
	// Load email template
	tmpl, err := template.ParseFiles("templates/reset_password_template.html")
	if err != nil {
		return err
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, struct{ OTP string }{OTP: otp}); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", "Restaurant System <"+os.Getenv("SMTP_EMAIL")+">")
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Reset Password OTP")
	m.SetBody("text/html", body.String())

	smtpPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		return err
	}

	d := gomail.NewDialer(
		os.Getenv("SMTP_HOST"),
		smtpPort,
		os.Getenv("SMTP_EMAIL"),
		os.Getenv("SMTP_PASSWORD"),
	)

	return d.DialAndSend(m)

}

// func QueueResetPasswordEmail(email, otp string) error {
//     // Implement email queuing logic here
//     // This could involve adding a job to a queue system or directly sending an email
//     // For now, we'll just print a message
//     fmt.Printf("Queued reset password email to %s with OTP: %s\n", email, otp)
//     return nil
// }

// // tasks/email.go
// package tasks

// import (
// 	"bytes"
// 	"encoding/json"
// 	"html/template"
// 	"log"
// 	"os"
// 	"strconv"

// 	"github.com/go-redis/redis/v8"
// 	"gopkg.in/gomail.v2"
// )

// var RedisClient *redis.Client

// func init() {
// 	RedisClient = redis.NewClient(&redis.Options{
// 		Addr: os.Getenv("REDIS_URL"),
// 	})
// }

// type EmailTask struct {
// 	Email string `json:"email"`
// 	OTP   string `json:"otp"`
// }

// func QueueVerificationEmail(email, otp string) error {
// 	task := EmailTask{
// 		Email: email,
// 		OTP:   otp,
// 	}

// 	jsonTask, err := json.Marshal(task)
// 	if err != nil {
// 		return err
// 	}

// 	err = RedisClient.RPush(RedisClient.Context(), "email_queue", jsonTask).Err()
// 	if err != nil {
// 		return err
// 	}

// 	log.Printf("Verification email for %s queued", email)
// 	return nil
// }

// func ProcessEmailQueue() {
// 	for {
// 		result, err := RedisClient.BLPop(RedisClient.Context(), 0, "email_queue").Result()
// 		if err != nil {
// 			log.Printf("Error popping from queue: %v", err)
// 			continue
// 		}

// 		var task EmailTask
// 		err = json.Unmarshal([]byte(result[1]), &task)
// 		if err != nil {
// 			log.Printf("Error unmarshaling task: %v", err)
// 			continue
// 		}

// 		err = sendEmail(task.Email, task.OTP)
// 		if err != nil {
// 			log.Printf("Error sending email: %v", err)
// 		}
// 	}
// }

// func sendEmail(email, otp string) error {
// 	// Load email template
// 	tmpl, err := template.ParseFiles("templates/email_template.html")
// 	if err != nil {
// 		return err
// 	}

// 	var body bytes.Buffer
// 	if err := tmpl.Execute(&body, struct{ OTP string }{OTP: otp}); err != nil {
// 		return err
// 	}

// 	m := gomail.NewMessage()
// 	m.SetHeader("From", "Restaurant System <"+os.Getenv("SMTP_EMAIL")+">")
// 	m.SetHeader("To", email)
// 	m.SetHeader("Subject", "Account Verification")
// 	m.SetBody("text/html", body.String())

// 	smtpPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
// 	if err != nil {
// 		return err
// 	}

// 	d := gomail.NewDialer(
// 		os.Getenv("SMTP_HOST"),
// 		smtpPort,
// 		os.Getenv("SMTP_EMAIL"),
// 		os.Getenv("SMTP_PASSWORD"),
// 	)

// 	return d.DialAndSend(m)
// }

// func QueueResetPasswordEmail(email, otp string) error {
// 	task := EmailTask{
// 		Email: email,
// 		OTP:   otp,
// 	}

// 	jsonTask, err := json.Marshal(task)
// 	if err != nil {
// 		return err
// 	}

// 	err = RedisClient.RPush(RedisClient.Context(), "email_queue", jsonTask).Err()
// 	if err != nil {
// 		return err
// 	}

// 	log.Printf("Reset password email for %s queued", email)
// 	return nil
// }
