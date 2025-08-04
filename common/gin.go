package common

import (
	"go-clean-arch/domain"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

type ClientInfo struct {
	UserAgent string
	IPAddress string
}

// ExtractClientInfo extracts client information from the Gin context
func ExtractClientInfo(c *gin.Context) *ClientInfo {
	return &ClientInfo{
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: GetClientIP(c),
	}
}

// GetClientIP gets the real client IP address
func GetClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header (for proxies)
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" && net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	xri := c.GetHeader("X-Real-IP")
	if xri != "" && net.ParseIP(xri) != nil {
		return xri
	}

	// Fallback to remote address
	remoteIP, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return remoteIP
}

// PopulateClientInfo automatically fills empty client info fields with extracted values
func PopulateClientInfo(c *gin.Context, ipAddress, userAgent *string) {
	clientInfo := ExtractClientInfo(c)

	if ipAddress != nil && *ipAddress == "" {
		*ipAddress = clientInfo.IPAddress
	}
	if userAgent != nil && *userAgent == "" {
		*userAgent = clientInfo.UserAgent
	}
}

func GetUserFromCtx(c *gin.Context) *domain.User {
	var userFromCtx *domain.User
	if v, ok := c.Get(UserContextKey); ok {
		if user, ok := v.(*domain.User); ok {
			userFromCtx = user
		}
	}

	return userFromCtx
}

func GetSessionIDFromCtx(c *gin.Context) string {
	var sIDFromCtx string
	if v, ok := c.Get(SessionIDContextKey); ok {
		if sID, ok := v.(string); ok {
			sIDFromCtx = sID
		}
	}
	return sIDFromCtx
}
