package services

import (
	"fmt"
	"strconv"
)

// DiscordInviteURL is the public Chorus community server.
const DiscordInviteURL = "https://discord.gg/7DVwM6jsS"

const (
	emailBanner = `<div style="background:#6366f1;border-radius:10px 10px 0 0;padding:28px 24px;text-align:center">
		<h1 style="color:#fff;margin:0;font-size:22px">Chorus</h1></div>`
	emailWrapA = `<table cellpadding="0" cellspacing="0" style="width:100%;background:#f4f4f6;padding:24px"><tr><td>
		<div style="max-width:520px;margin:0 auto;background:#ffffff;border-radius:10px;overflow:hidden;font-family:Arial,Helvetica,sans-serif;color:#1f2937">`
	emailWrapB = `</div></td></tr></table>`
	emailButton = `<p style="margin:18px 0"><a href="%s" style="display:inline-block;background:#6366f1;color:#ffffff;padding:12px 20px;border-radius:8px;text-decoration:none;font-weight:600">%s</a></p>`
	emailText   = `<p style="margin:0 0 10px 0">%s</p>`
	emailFooter = `<div style="padding:18px 24px;color:#6b7280;font-size:12px;border-top:1px solid #eef0f3">Chorus · Break language barriers.<br>Questions? info@chorus.talk · Join us on <a href="` + DiscordInviteURL + `" style="color:#6366f1">Discord</a></div>`
)

func wrapHTML(content string) string {
	return `<html><body style="margin:0;background:#f4f4f6">` + emailWrapA + emailBanner +
		`<div style="padding:24px">` + content + `</div>` + emailFooter + emailWrapB + `</body></html>`
}

// WaitlistConfirmationEmail returns the subject and HTML body for a fresh signup.
func WaitlistConfirmationEmail(queuePosition int) (string, string) {
	subject := "You're on the Chorus waitlist"
	content := fmt.Sprintf(emailText, "Thanks for joining the Chorus waitlist! Your place in line is <strong>#"+strconv.Itoa(queuePosition)+"</strong>.") +
		fmt.Sprintf(emailText, "We're rolling out in batches. When your spot comes up we'll email you a private sign-up link, so keep an eye on your inbox (and spam folder).") +
		fmt.Sprintf(emailText, "In the meantime, connect with the Chorus community:") +
		fmt.Sprintf(emailButton, DiscordInviteURL, "Join the Chorus Discord") +
		"<p style=\"font-size:12px;color:#6b7280\">Stay tuned!</p>"
	return subject, wrapHTML(content)
}

// UpdatedWaitlistConfirmationEmail is sent when the user re-submits the form
// and their preferences were refreshed while keeping their queue position.
func UpdatedWaitlistConfirmationEmail(queuePosition int) (string, string) {
	subject := "We've updated your Chorus waitlist preferences"
	content := fmt.Sprintf(emailText, "You recently signed up for the Chorus waitlist again, so we refreshed your preferences. Your spot in line <strong>#" + strconv.Itoa(queuePosition) + "</strong> is still yours.") +
		fmt.Sprintf(emailText, "No need to do anything — we'll email you a private sign-up link when your turn comes.") +
		fmt.Sprintf(emailText, "In the meantime, connect with the Chorus community:") +
		fmt.Sprintf(emailButton, DiscordInviteURL, "Join the Chorus Discord") +
		"<p style=\"font-size:12px;color:#6b7280\">Stay tuned!</p>"
	return subject, wrapHTML(content)
}

// InvitationEmail returns the subject and HTML body for an admin-approved user
// who should now create their account via the provided signup link.
func InvitationEmail(signupLink string) (string, string) {
	subject := "You're invited to Chorus!"
	content := fmt.Sprintf(emailText, "Good news — you've moved off the Chorus waitlist and you're invited to create an account!") +
		fmt.Sprintf(emailButton, signupLink, "Create your account") +
		fmt.Sprintf(emailText, "This link is private and single-use, and it's tied to your email address.") +
		fmt.Sprintf(emailText, "Join the community while you're at it:") +
		fmt.Sprintf(`<p style="margin:0 0 10px 0"><a href="`+DiscordInviteURL+`">Join the Chorus Discord</a></p>`) +
		fmt.Sprintf(emailText, "If the button doesn't work, copy and paste this link into your browser: "+signupLink) +
		fmt.Sprintf(emailText, "Welcome to Chorus!")
	return subject, wrapHTML(content)
}

// PasswordResetEmail returns the subject and HTML body for the forgot-password
// flow. The link is valid for 60 minutes and single-use.
func PasswordResetEmail(resetLink string) (string, string) {
	subject := "Reset your Chorus password"
	content := fmt.Sprintf(emailText, "We received a request to reset your Chorus password.") +
		fmt.Sprintf(emailText, "If this was you, click the button below to choose a new password:") +
		fmt.Sprintf(emailButton, resetLink, "Reset your password") +
		fmt.Sprintf(emailText, "This link is valid for 60 minutes and can only be used once.") +
		fmt.Sprintf(emailText, "If you didn't request this, you can safely ignore this email — your password won't change.") +
		fmt.Sprintf(emailText, "If the button doesn't work, copy and paste this link into your browser: "+resetLink)
	return subject, wrapHTML(content)
}

// RegistrationWelcomeEmail returns the subject and HTML body sent to a newly
// registered user.
func RegistrationWelcomeEmail(displayName string) (string, string) {
	subject := "Welcome to Chorus!"
	content := fmt.Sprintf(emailText, "Hi "+displayName+", welcome to Chorus!") +
		fmt.Sprintf(emailText, "You can now send messages across languages with built-in translation, grammar help, and more.") +
		fmt.Sprintf(emailText, "Start a conversation and explore what Chorus can do:") +
		fmt.Sprintf(emailButton, "https://chorus.talk", "Open Chorus") +
		fmt.Sprintf(emailText, "If you have questions, we're on Discord:") +
		fmt.Sprintf(`<p style="margin:0 0 10px 0"><a href="`+DiscordInviteURL+`">Join the Chorus Discord</a></p>`)
	return subject, wrapHTML(content)
}