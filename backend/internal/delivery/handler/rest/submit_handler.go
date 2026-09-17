package rest

import (
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"

	"dpdp-backend/internal/delivery"
	smtphandler "dpdp-backend/internal/delivery/handler/smtp"
	deliveryutils "dpdp-backend/internal/delivery/utils"
)

type SubmitHandler struct {
	authorizer *smtphandler.Authorizer
	acceptor   *delivery.Acceptor
	maxSize    int64
	maxRcpt    int
}

func NewSubmitHandler(
	authorizer *smtphandler.Authorizer,
	acceptor *delivery.Acceptor,
	maxSize int64,
	maxRecipients int,
) *SubmitHandler {
	return &SubmitHandler{
		authorizer: authorizer,
		acceptor:   acceptor,
		maxSize:    maxSize,
		maxRcpt:    maxRecipients,
	}
}

func (h *SubmitHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/delivery/messages", h.submit)
}

func (h *SubmitHandler) submit(c *gin.Context) {
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, h.maxSize+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request body could not be read"})
		return
	}

	if int64(len(raw)) > h.maxSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "message exceeds maximum size"})
		return
	}

	parsed, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message could not be parsed"})
		return
	}

	from := addressOf(parsed.Header.Get("From"))
	if from == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a From header is required"})
		return
	}

	domain := delivery.DomainOf(from)
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "malformed sender address"})
		return
	}

	recipients := recipientsOf(parsed.Header)

	if len(recipients) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one To or Cc recipient is required"})
		return
	}

	if len(recipients) > h.maxRcpt {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many recipients"})
		return
	}

	authorization, err := h.authorizer.Authorize(c.Request.Context(), domain)
	if err != nil {
		if errors.Is(err, deliveryutils.ErrDomainUnknown) {
			c.JSON(http.StatusForbidden, gin.H{"error": "sender domain not authorized"})
			return
		}

		c.Header("Retry-After", "30")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sender domain lookup unavailable"})

		return
	}

	msg, err := h.acceptor.Accept(c.Request.Context(), delivery.Submission{
		CustomerID:   authorization.CustomerID,
		ConfigID:     authorization.ConfigID,
		From:         from,
		SenderDomain: domain,
		Recipients:   recipients,
		Raw:          raw,
	})
	if err != nil {
		c.Header("Retry-After", "30")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"correlation_id": msg.CorrelationID,
		"message_id":     msg.MessageID,
		"recipients":     msg.Recipients,
		"size":           msg.Size,
	})
}

func recipientsOf(header mail.Header) []string {
	recipients := make([]string, 0)
	seen := map[string]bool{}

	for _, field := range []string{"To", "Cc"} {
		for _, value := range header[field] {
			addresses, err := mail.ParseAddressList(value)
			if err != nil {
				continue
			}

			for _, address := range addresses {
				if delivery.DomainOf(address.Address) == "" || seen[address.Address] {
					continue
				}

				seen[address.Address] = true
				recipients = append(recipients, address.Address)
			}
		}
	}

	return recipients
}

func addressOf(value string) string {
	address, err := mail.ParseAddress(value)
	if err != nil {
		return strings.TrimSpace(value)
	}

	return address.Address
}
