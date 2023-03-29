package errors

import (
	"fmt"
	"go-example/internal/dto"
	"go-example/internal/log"
	"net/http"

	"github.com/gin-gonic/gin"
)

var ErrReplyUnknown = dto.ReplyError("Unknown error")

const tagUnhandlerError = "[UnhandlerError]:"
const tagAppError = "[AppError]:"

// GinError middleware
func GinError() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if errors := c.Errors.ByType(gin.ErrorTypeAny); len(errors) > 0 {
			err := errors[0].Err
			if err, ok := err.(*Error); ok {
				log.Error(
					fmt.Sprintf("%s: %s", tagAppError, err))
				c.AbortWithStatusJSON(err.Code, err.ToReply())
				return
			}
			log.Error(
				fmt.Sprintf("%s: %s", tagUnhandlerError, err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, ErrReplyUnknown)
			return
		}
	}
}
