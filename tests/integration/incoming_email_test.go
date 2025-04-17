// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"encoding/base32"
	"io"
	"net"
	"net/smtp"
	"strings"
	"testing"
	"time"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/services/mailer/incoming"
	incoming_payload "forgejo.org/services/mailer/incoming/payload"
	token_service "forgejo.org/services/mailer/token"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/gomail.v2"
)

func TestIncomingEmail(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})

	t.Run("Payload", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 1})

		_, err := incoming_payload.CreateReferencePayload(user)
		require.Error(t, err)

		issuePayload, err := incoming_payload.CreateReferencePayload(issue)
		require.NoError(t, err)
		commentPayload, err := incoming_payload.CreateReferencePayload(comment)
		require.NoError(t, err)

		_, err = incoming_payload.GetReferenceFromPayload(db.DefaultContext, []byte{1, 2, 3})
		require.Error(t, err)

		ref, err := incoming_payload.GetReferenceFromPayload(db.DefaultContext, issuePayload)
		require.NoError(t, err)
		assert.IsType(t, ref, new(issues_model.Issue))
		assert.Equal(t, issue.ID, ref.(*issues_model.Issue).ID)

		ref, err = incoming_payload.GetReferenceFromPayload(db.DefaultContext, commentPayload)
		require.NoError(t, err)
		assert.IsType(t, ref, new(issues_model.Comment))
		assert.Equal(t, comment.ID, ref.(*issues_model.Comment).ID)
	})

	t.Run("Token", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		payload := []byte{1, 2, 3, 4, 5}

		token, err := token_service.CreateToken(token_service.ReplyHandlerType, user, payload)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		ht, u, p, err := token_service.ExtractToken(db.DefaultContext, token)
		require.NoError(t, err)
		assert.Equal(t, token_service.ReplyHandlerType, ht)
		assert.Equal(t, user.ID, u.ID)
		assert.Equal(t, payload, p)
	})

	tokenEncoding := base32.StdEncoding.WithPadding(base32.NoPadding)
	t.Run("Deprecated token version", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		payload := []byte{1, 2, 3, 4, 5}

		token, err := token_service.CreateToken(token_service.ReplyHandlerType, user, payload)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Set the token to version 1.
		unencodedToken, err := tokenEncoding.DecodeString(token)
		require.NoError(t, err)
		unencodedToken[0] = 1
		token = tokenEncoding.EncodeToString(unencodedToken)

		ht, u, p, err := token_service.ExtractToken(db.DefaultContext, token)
		require.ErrorContains(t, err, "unsupported token version: 1")
		assert.Equal(t, token_service.UnknownHandlerType, ht)
		assert.Nil(t, u)
		assert.Nil(t, p)
	})

	t.Run("MAC check", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		payload := []byte{1, 2, 3, 4, 5}

		token, err := token_service.CreateToken(token_service.ReplyHandlerType, user, payload)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Modify the MAC.
		unencodedToken, err := tokenEncoding.DecodeString(token)
		require.NoError(t, err)
		unencodedToken[len(unencodedToken)-1] ^= 0x01
		token = tokenEncoding.EncodeToString(unencodedToken)

		ht, u, p, err := token_service.ExtractToken(db.DefaultContext, token)
		require.ErrorContains(t, err, "verification failed")
		assert.Equal(t, token_service.UnknownHandlerType, ht)
		assert.Nil(t, u)
		assert.Nil(t, p)
	})

	t.Run("Handler", func(t *testing.T) {
		t.Run("Reply", func(t *testing.T) {
			checkReply := func(t *testing.T, payload []byte, issue *issues_model.Issue, commentType issues_model.CommentType) {
				t.Helper()

				handler := &incoming.ReplyHandler{}

				require.Error(t, handler.Handle(db.DefaultContext, &incoming.MailContent{}, nil, payload))
				require.NoError(t, handler.Handle(db.DefaultContext, &incoming.MailContent{}, user, payload))

				content := &incoming.MailContent{
					Content: "reply by mail",
					Attachments: []*incoming.Attachment{
						{
							Name:    "attachment.txt",
							Content: []byte("test"),
						},
					},
				}

				require.NoError(t, handler.Handle(db.DefaultContext, content, user, payload))

				comments, err := issues_model.FindComments(db.DefaultContext, &issues_model.FindCommentsOptions{
					IssueID: issue.ID,
					Type:    commentType,
				})
				require.NoError(t, err)
				assert.NotEmpty(t, comments)
				comment := comments[len(comments)-1]
				assert.Equal(t, user.ID, comment.PosterID)
				assert.Equal(t, content.Content, comment.Content)
				require.NoError(t, comment.LoadAttachments(db.DefaultContext))
				assert.Len(t, comment.Attachments, 1)
				attachment := comment.Attachments[0]
				assert.Equal(t, content.Attachments[0].Name, attachment.Name)
				assert.EqualValues(t, 4, attachment.Size)
			}
			t.Run("Issue", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				payload, err := incoming_payload.CreateReferencePayload(issue)
				require.NoError(t, err)

				checkReply(t, payload, issue, issues_model.CommentTypeComment)
			})

			t.Run("CodeComment", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 6})
				issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})

				payload, err := incoming_payload.CreateReferencePayload(comment)
				require.NoError(t, err)

				checkReply(t, payload, issue, issues_model.CommentTypeCode)
			})

			t.Run("Comment", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2})
				issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})

				payload, err := incoming_payload.CreateReferencePayload(comment)
				require.NoError(t, err)

				checkReply(t, payload, issue, issues_model.CommentTypeComment)
			})
		})

		t.Run("Unsubscribe", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			watching, err := issues_model.CheckIssueWatch(db.DefaultContext, user, issue)
			require.NoError(t, err)
			assert.True(t, watching)

			handler := &incoming.UnsubscribeHandler{}

			content := &incoming.MailContent{
				Content: "unsub me",
			}

			payload, err := incoming_payload.CreateReferencePayload(issue)
			require.NoError(t, err)

			require.NoError(t, handler.Handle(db.DefaultContext, content, user, payload))

			watching, err = issues_model.CheckIssueWatch(db.DefaultContext, user, issue)
			require.NoError(t, err)
			assert.False(t, watching)
		})
	})

	if setting.IncomingEmail.Enabled {
		// This test connects to the configured email server and is currently only enabled for MySql integration tests.
		// It sends a reply to create a comment. If the comment is not detected after 10 seconds the test fails.
		t.Run("IMAP", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			payload, err := incoming_payload.CreateReferencePayload(issue)
			require.NoError(t, err)
			token, err := token_service.CreateToken(token_service.ReplyHandlerType, user, payload)
			require.NoError(t, err)

			msg := gomail.NewMessage()
			msg.SetHeader("To", strings.Replace(setting.IncomingEmail.ReplyToAddress, setting.IncomingEmail.TokenPlaceholder, token, 1))
			msg.SetHeader("From", user.Email)
			msg.SetBody("text/plain", token)
			err = gomail.Send(&smtpTestSender{}, msg)
			require.NoError(t, err)

			assert.Eventually(t, func() bool {
				comments, err := issues_model.FindComments(db.DefaultContext, &issues_model.FindCommentsOptions{
					IssueID: issue.ID,
					Type:    issues_model.CommentTypeComment,
				})
				require.NoError(t, err)
				assert.NotEmpty(t, comments)

				comment := comments[len(comments)-1]

				return comment.PosterID == user.ID && comment.Content == token
			}, 10*time.Second, 1*time.Second)
		})
	}
}

// A simple SMTP mail sender used for integration tests.
type smtpTestSender struct{}

func (s *smtpTestSender) Send(from string, to []string, msg io.WriterTo) error {
	conn, err := net.Dial("tcp", net.JoinHostPort(setting.IncomingEmail.Host, "25"))
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, setting.IncomingEmail.Host)
	if err != nil {
		return err
	}

	if err = client.Mail(from); err != nil {
		return err
	}

	for _, rec := range to {
		if err = client.Rcpt(rec); err != nil {
			return err
		}
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := msg.WriteTo(w); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	return client.Quit()
}
