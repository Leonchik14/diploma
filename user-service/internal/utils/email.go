package utils

import (
	"fmt"
	"math/rand"
	"net/smtp"
	"time"
)

func SendPasswordResetCode(email, code, smtpHost, smtpPort, smtpUser, smtpPassword, fromEmail, fromName string) error {
	// Настройка SMTP
	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)

	// Формирование письма
	to := []string{email}
	subject := "Код восстановления пароля"
	body := fmt.Sprintf(`
Привет!

Вы запросили восстановление пароля для вашего аккаунта.

Ваш код восстановления: %s

Этот код действителен в течение 15 минут.

Если вы не запрашивали восстановление пароля, просто проигнорируйте это письмо.

С уважением,
Команда %s
`, code, fromName)

	message := fmt.Sprintf("From: %s <%s>\r\n", fromName, fromEmail)
	message += fmt.Sprintf("To: %s\r\n", email)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/plain; charset=UTF-8\r\n"
	message += "\r\n" + body

	// Отправка письма
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	err := smtp.SendMail(addr, auth, fromEmail, to, []byte(message))
	return err
}

// Генерирует 6-значный код
func GenerateResetCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}
