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
		SMTPHost:     GetEnvString("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     GetEnvInt("SMTP_PORT", 587),
		SMTPUsername: GetEnvString("SMTP_USERNAME", "khorzehung@gmail.com"),
		SMTPPassword: GetEnvString("SMTP_PASSWORD", "eqyu mium udby hvic"),
		FromEmail:    GetEnvString("FROM_EMAIL", "khorzehung@gmai.com"),
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
		GetEnvString("FRONTEND_URL", "https://app.renotech.space"),
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
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            line-height: 1.6; 
            color: #2c3e50; 
            margin: 0; 
            padding: 0; 
            background: linear-gradient(135deg, #e3f2fd 0%, #bbdefb 100%);
        }
        .container { 
            max-width: 600px; 
            margin: 0 auto; 
            padding: 20px; 
            background-color: #ffffff;
            border-radius: 12px;
            box-shadow: 0 8px 32px rgba(33, 150, 243, 0.1);
        }
        .header { 
            background: linear-gradient(135deg, #2196f3 0%, #1976d2 100%); 
            padding: 30px 20px; 
            text-align: center; 
            border-radius: 12px 12px 0 0;
            margin: -20px -20px 0 -20px;
        }
        .header h1 {
            color: #ffffff;
            margin: 0;
            font-size: 28px;
            font-weight: 600;
            text-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
        }
        .content { 
            padding: 30px 20px; 
            background-color: #ffffff; 
        }
        .content h2 {
            color: #1976d2;
            margin-top: 0;
            font-size: 24px;
            font-weight: 500;
        }
        .content p {
            color: #37474f;
            font-size: 16px;
            margin: 16px 0;
        }
        .button { 
            display: inline-block; 
            padding: 16px 32px; 
            background: linear-gradient(135deg, #2196f3 0%, #1976d2 100%); 
            color: white; 
            text-decoration: none; 
            border-radius: 8px; 
            margin: 24px 0; 
            font-weight: 600;
            font-size: 16px;
            box-shadow: 0 4px 16px rgba(33, 150, 243, 0.3);
            transition: all 0.3s ease;
        }
        .button:hover {
            background: linear-gradient(135deg, #1976d2 0%, #1565c0 100%);
            box-shadow: 0 6px 20px rgba(33, 150, 243, 0.4);
        }
        .link-box {
            word-break: break-all; 
            background: linear-gradient(135deg, #e3f2fd 0%, #f3e5f5 100%); 
            padding: 16px; 
            border-radius: 8px; 
            border: 2px solid #bbdefb;
            margin: 16px 0;
            font-family: 'Courier New', monospace;
            font-size: 14px;
            color: #1565c0;
        }
        .footer { 
            background: linear-gradient(135deg, #eceff1 0%, #cfd8dc 100%); 
            padding: 20px; 
            text-align: center; 
            font-size: 12px; 
            color: #607d8b; 
            border-radius: 0 0 12px 12px;
            margin: 0 -20px -20px -20px;
        }
        .warning { 
            background: linear-gradient(135deg, #e1f5fe 0%, #b3e5fc 100%); 
            border: 2px solid #29b6f6; 
            padding: 16px; 
            border-radius: 8px; 
            margin: 20px 0; 
            color: #0277bd;
            font-weight: 500;
        }
        .warning strong {
            color: #01579b;
        }
        .divider {
            height: 2px;
            background: linear-gradient(90deg, #e3f2fd 0%, #2196f3 50%, #e3f2fd 100%);
            margin: 24px 0;
            border: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔧 RenoTech - Password Reset</h1>
        </div>
        <div class="content">
            <h2>🔐 Reset Your Password</h2>
            <p>Hello there! 👋</p>
            <p>We received a request to reset your password for your RenoTech account associated with this email address.</p>
            
            <hr class="divider">
            
            <p>Click the button below to reset your password:</p>
            <center>
                <a href="{{.ResetLink}}" class="button">🔄 Reset Password</a>
            </center>
            
            <p>Or copy and paste this link into your browser:</p>
            <div class="link-box">{{.ResetLink}}</div>
            
            <div class="warning">
                <strong>⚠️ Important:</strong> This link will expire in 5 minutes for security reasons.
            </div>
            
            <hr class="divider">
            
            <p>If you didn't request this password reset, please ignore this email or contact support if you have concerns.</p>
            <p>Best regards,<br><strong>The RenoTech Team</strong> 🏗️</p>
        </div>
        <div class="footer">
            <p>📧 This is an automated message. Please do not reply to this email.</p>
            <p>© 2025 RenoTech. All rights reserved.</p>
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
