package utils

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
)

var (
	emailFrom     = os.Getenv("EMAIL_FROM")
	emailPassword = os.Getenv("EMAIL_PASSWORD")
	smtpHost      = os.Getenv("SMTP_HOST")
	smtpPort      = os.Getenv("SMTP_PORT")
	companyName   = "MooveIt Limited"
	baseURL       = os.Getenv("BASE_URL")
)

// Common header template for all emails
const emailHeader = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0;">
	<div style="max-width: 600px; margin: 0 auto; padding: 20px;">
		<div style="text-align: center; margin-bottom: 30px; background-color: #f9f9f9; padding: 20px;">
			<!-- <img src="%s/static/images/logo.png" alt="MooveIt" style="width: 200px; height: auto;"> -->
			<h2 style="color: #4CAF50; margin: 0;">MooveIt</h2>
		</div>
`

// Common footer template for all emails
const emailFooter = `
		<div style="text-align: center; margin-top: 20px; font-size: 12px; color: #666; border-top: 1px solid #eee; padding-top: 20px;">
			<p>This is an automated message, please do not reply to this email.</p>
			<p>© 2025 MooveIt Limited. All rights reserved.</p>
		</div>
	</div>
</body>
</html>
`

func sendEmail(to []string, subject, body string) error {
	if emailFrom == "" || emailPassword == "" || smtpHost == "" || smtpPort == "" {
		return fmt.Errorf("email configuration not set")
	}

	// Headers
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", companyName, emailFrom)
	headers["To"] = strings.Join(to, ",")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"
	headers["X-Priority"] = "1"
	headers["X-MSMail-Priority"] = "High"
	headers["X-Mailer"] = "MooveIt-Mailer"
	headers["List-Unsubscribe"] = fmt.Sprintf("<%s>", emailFrom)

	// Build message
	message := ""
	for key, value := range headers {
		message += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	message += "\r\n" + body

	// Authentication
	auth := smtp.PlainAuth("", emailFrom, emailPassword, smtpHost)

	// Send email
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, emailFrom, to, []byte(message))
	if err != nil {
		log.Printf("Failed to send email: %v", err)
		return err
	}

	log.Printf("Successfully sent email to recipients: %v", to)
	return nil
}

func SendNewBookingNotificationEmailToDriver(driverEmail, destination, clientName string) error {
	subject := "New Booking Request - MooveIt"
	body := fmt.Sprintf(emailHeader+`
				<div style="background-color: #f9f9f9; padding: 20px; border-radius: 5px;">
					<h1 style="color: #2c3e50; text-align: center;">New Booking Request</h1>
					<p>Hello,</p>
					<p>You have received a new booking request for your ride to <strong>%s</strong> from <strong>%s</strong>.</p>
					<p>Please log in to your MooveIt account to accept or reject this booking.</p>
					<div style="text-align: center; margin: 30px 0;">
						<a href="%s/login" style="background-color: #4CAF50; color: white; padding: 12px 25px; text-decoration: none; border-radius: 5px;">Login to MooveIt</a>
					</div>
					<p>Best regards,<br>The MooveIt Team</p>
				</div>`+emailFooter,
		baseURL, destination, clientName, baseURL)

	return sendEmail([]string{driverEmail}, subject, body)
}

func SendBookingAcceptedEmail(clientEmail, driverName, carPlate, receiverEmail, receiverName string) error {
	// Email to client
	clientSubject := "Booking Accepted - MooveIt"
	clientBody := fmt.Sprintf(emailHeader+`
				<div style="background-color: #f9f9f9; padding: 20px; border-radius: 5px;">
					<h1 style="color: #2c3e50; text-align: center;">Booking Accepted</h1>
					<p>Hello,</p>
					<p>Great news! Your booking has been accepted by driver <strong>%s</strong> (Car: <strong>%s</strong>).</p>
					<p>Your parcel is now ready for delivery. You will receive updates about your delivery status.</p>
					<div style="text-align: center; margin: 30px 0;">
						<a href="%s/tracking" style="background-color: #4CAF50; color: white; padding: 12px 25px; text-decoration: none; border-radius: 5px;">Track Your Parcel</a>
					</div>
					<p>Best regards,<br>The MooveIt Team</p>
				</div>`+emailFooter,
		baseURL, driverName, carPlate, baseURL)

	if err := sendEmail([]string{clientEmail}, clientSubject, clientBody); err != nil {
		return fmt.Errorf("failed to send email to client: %v", err)
	}

	// Email to receiver
	receiverSubject := "Incoming Parcel Delivery - MooveIt"
	receiverBody := fmt.Sprintf(emailHeader+`
				<div style="background-color: #f9f9f9; padding: 20px; border-radius: 5px;">
					<h1 style="color: #2c3e50; text-align: center;">Incoming Parcel Delivery</h1>
					<p>Hello %s,</p>
					<p>A parcel is being delivered to you by <strong>%s</strong> (Car: <strong>%s</strong>).</p>
					<p>You will be notified when the parcel arrives at your location.</p>
					<div style="text-align: center; margin: 30px 0;">
						<a href="%s/tracking" style="background-color: #4CAF50; color: white; padding: 12px 25px; text-decoration: none; border-radius: 5px;">Track Your Parcel</a>
					</div>
					<p>Best regards,<br>The MooveIt Team</p>
				</div>`+emailFooter,
		baseURL, receiverName, driverName, carPlate, baseURL)

	if err := sendEmail([]string{receiverEmail}, receiverSubject, receiverBody); err != nil {
		return fmt.Errorf("failed to send email to receiver: %v", err)
	}

	return nil
}

func SendBookingRejectedEmail(clientEmail string) error {
	subject := "Booking Rejected - MooveIt"
	body := fmt.Sprintf(emailHeader+`
				<div style="background-color: #f9f9f9; padding: 20px; border-radius: 5px;">
					<h1 style="color: #2c3e50; text-align: center;">Booking Rejected</h1>
					<p>Hello,</p>
					<p>Unfortunately, your booking has been rejected by the driver.</p>
					<p>Don't worry! You can try booking another available ride.</p>
					<div style="text-align: center; margin: 30px 0;">
						<a href="%s/rides" style="background-color: #4CAF50; color: white; padding: 12px 25px; text-decoration: none; border-radius: 5px;">Find Another Ride</a>
					</div>
					<p>Best regards,<br>The MooveIt Team</p>
				</div>`+emailFooter,
		baseURL)
	return sendEmail([]string{clientEmail}, subject, body)
} 