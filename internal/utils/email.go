package utils

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"renotech.com.my/internal/enum"
)

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

type EmailData struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

func GetEmailConfig() *EmailConfig {
	return &EmailConfig{
		SMTPHost:     GetEnvString("SMTP_HOST", "localhost"),
		SMTPPort:     GetEnvInt("SMTP_PORT", 587),
		SMTPUsername: GetEnvString("SMTP_USERNAME", ""),
		SMTPPassword: GetEnvString("SMTP_PASSWORD", ""),
		FromEmail:    GetEnvString("FROM_EMAIL", "noreply@renotech.com.my"),
		FromName:     GetEnvString("FROM_NAME", "RenoTech"),
	}
}

func SendEmail(emailData *EmailData) error {
	config := GetEmailConfig()

	if config.SMTPHost == "localhost" || config.SMTPUsername == "" {
		return SystemError(enum.ErrorCodeInternal, "Email configuration not set up", nil)
	}

	auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPHost)

	from := fmt.Sprintf("%s <%s>", config.FromName, config.FromEmail)
	to := []string{emailData.To}

	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msgBuilder.WriteString(fmt.Sprintf("To: %s\r\n", emailData.To))
	msgBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", emailData.Subject))

	if emailData.IsHTML {
		msgBuilder.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		msgBuilder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(emailData.Body)

	msg := []byte(msgBuilder.String())

	smtpAddr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
	err := smtp.SendMail(smtpAddr, auth, config.FromEmail, to, msg)
	if err != nil {
		return SystemError(enum.ErrorCodeInternal, "Failed to send email", map[string]interface{}{
			"error": err.Error(),
		})
	}

	return nil
}

func SendPasswordResetEmail(email, resetToken string) error {
	resetLink := fmt.Sprintf("%s/reset-password?token=%s&email=%s",
		GetEnvString("FRONTEND_URL", "http://localhost:3000"),
		resetToken,
		email,
	)

	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #f4f4f4; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #ffffff; }
        .button { display: inline-block; padding: 10px 20px; background-color: #007bff; color: white; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { background-color: #f4f4f4; padding: 10px; text-align: center; font-size: 12px; color: #666; }
        .warning { background-color: #fff3cd; border: 1px solid #ffeaa7; padding: 10px; border-radius: 4px; margin: 15px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>RenoTech - Password Reset</h1>
        </div>
        <div class="content">
            <h2>Reset Your Password</h2>
            <p>Hello,</p>
            <p>We received a request to reset your password for your RenoTech account associated with this email address.</p>
            <p>Click the button below to reset your password:</p>
            <a href="{{.ResetLink}}" class="button">Reset Password</a>
            <p>Or copy and paste this link into your browser:</p>
            <p style="word-break: break-all; background-color: #f8f9fa; padding: 10px; border-radius: 4px;">{{.ResetLink}}</p>
            <div class="warning">
                <strong>Important:</strong> This link will expire in 24 hours for security reasons.
            </div>
            <p>If you didn't request this password reset, please ignore this email or contact support if you have concerns.</p>
            <p>Best regards,<br>The RenoTech Team</p>
        </div>
        <div class="footer">
            <p>This is an automated message. Please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>`

	tmpl, err := template.New("passwordReset").Parse(htmlTemplate)
	if err != nil {
		return SystemError(enum.ErrorCodeInternal, "Failed to parse email template", nil)
	}

	var body bytes.Buffer
	err = tmpl.Execute(&body, map[string]string{
		"ResetLink": resetLink,
	})
	if err != nil {
		return SystemError(enum.ErrorCodeInternal, "Failed to generate email content", nil)
	}

	emailData := &EmailData{
		To:      email,
		Subject: "RenoTech - Password Reset Request",
		Body:    body.String(),
		IsHTML:  true,
	}

	return SendEmail(emailData)
}