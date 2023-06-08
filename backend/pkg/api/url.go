package api

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	db "github.com/ngtrdai197/url-shorterner/db/sqlc"
	"github.com/ngtrdai197/url-shorterner/pkg/token"
	"github.com/rs/zerolog/log"
)

type createUrlRequest struct {
	LongUrl     string `json:"long_url" binding:"required"`
	Description string `json:"description"`
}

type rediectUrlRequest struct {
	ShortUrl string `form:"v" binding:"required"`
}

type urlResponse struct {
	Id          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	ShortUrl    string    `json:"short_url"`
	LongUrl     string    `json:"long_url"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type getListUrlRequest struct {
	Limit int `form:"limit"`
	Page  int `form:"page"`
}

type paginateData struct {
	Limit int   `json:"limit"`
	Page  int   `json:"page"`
	Total int64 `json:"total"`
}

type getListUrlResponse struct {
	Paginate paginateData  `json:"paginate"`
	Data     []urlResponse `json:"data"`
}

func newUrlResponse(url db.Url) urlResponse {
	return urlResponse{
		Id:          url.ID,
		UserID:      url.UserID,
		ShortUrl:    url.ShortUrl,
		LongUrl:     url.LongUrl,
		CreatedAt:   url.CreatedAt,
		Description: url.Description.String,
	}
}

func (s *Server) CreateUrl(ctx *gin.Context) {
	var req createUrlRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	fmt.Printf("request %v", req)
	found, err := s.store.GetUrlByLong(ctx, req.LongUrl)
	if err == nil {
		ctx.JSON(http.StatusOK, newUrlResponse(found))
		return
	}

	if err == sql.ErrNoRows {
		node, err := snowflake.NewNode(97)
		if err != nil {
			log.Error().Msgf("new node error=%v", err)
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
		ID := node.Generate().Int64()
		arg := db.CreateUrlParams{
			ID:          ID,
			ShortUrl:    base64.RawURLEncoding.EncodeToString(big.NewInt(int64(ID)).Bytes()),
			LongUrl:     req.LongUrl,
			UserID:      authPayload.UserID,
			Description: sql.NullString{String: req.Description, Valid: true},
		}
		url, err := s.store.CreateUrl(ctx, arg)
		if err != nil {
			log.Error().Msgf("create url with error=%v", err)
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		response := newUrlResponse(url)
		ctx.JSON(http.StatusOK, response)
		return
	}
	ctx.JSON(http.StatusInternalServerError, errorResponse(err))
}

func (s *Server) GetListUrl(ctx *gin.Context) {
	req := getListUrlRequest{
		Limit: defaultLimit,
		Page:  defaultPage,
	}

	if err := ctx.BindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	urls, err := s.store.GetListURLs(ctx, db.GetListURLsParams{
		UserID: authPayload.UserID,
		Limit:  int32(req.Limit),
		Offset: int32((req.Page - 1) * req.Limit),
	})
	if err != nil && err != sql.ErrNoRows {
		if err == sql.ErrNoRows {
			urls = []db.Url{}
		} else {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	}

	totalUrls, err := s.store.GetCountURLs(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, getListUrlResponse{
		Paginate: paginateData{
			Limit: req.Limit,
			Page:  req.Page,
			Total: totalUrls,
		},
		Data: convertDBUrlsToURLResponses(urls),
	})
}

func (s *Server) RedirectUrl(ctx *gin.Context) {
	var req rediectUrlRequest

	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	url, err := s.store.GetUrlByShort(ctx, req.ShortUrl)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			ctx.JSON(http.StatusInternalServerError, errorResponse(pqErr))
		}
		ctx.JSON(http.StatusNotFound, errorResponse(err))
	}

	http.Redirect(ctx.Writer, ctx.Request, url.LongUrl, http.StatusSeeOther)
}

func convertDBUrlsToURLResponses(dbUrls []db.Url) []urlResponse {
	urlResponses := make([]urlResponse, len(dbUrls))

	for i, dbUrl := range dbUrls {
		urlResponses[i] = newUrlResponse(dbUrl)
	}

	return urlResponses
}
