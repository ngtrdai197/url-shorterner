package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/ngtrdai197/url-shortener/config"
	db "github.com/ngtrdai197/url-shortener/db/sqlc"
	"github.com/ngtrdai197/url-shortener/pkg/token"
	"github.com/rs/zerolog/log"
)

type Server struct {
	config     *config.Config
	store      db.Store
	router     *gin.Engine
	tokenMaker token.Maker
}

func NewServer(c *config.Config) *Server {
	conn, err := sql.Open(c.DbDriver, c.DbSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect db")
	}

	tokenMaker, err := token.NewPasetoMaker(c)
	if err != nil {
		log.Fatal().Err(err).Msgf("cannot create token maker: %v", err)
	}

	store := db.NewStore(conn)
	server := &Server{config: c, store: store, tokenMaker: tokenMaker}
	server.setupRouter()

	return server
}

func (s *Server) setupRouter() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
		log.Printf("endpoint %v %v %v %v", httpMethod, absolutePath, handlerName, nuHandlers)
	}
	r.Use(corsMiddleware())

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/pk", func(c *gin.Context) {
		c.JSON(http.StatusOK, transformApiResponse(SuccessCode, "ok", s.config.PublicKey))
	})

	r.GET("/r", s.RedirectUrl)

	authRoutes := r.Group("/auth")
	{
		authRoutes.POST("/login", s.loginUser)
		authRoutes.POST("/renew-token", s.renewAccessToken)
		authRoutes.POST("/register", s.createUser)
	}

	authenticatedRoutes := r.Group("/")
	authenticatedRoutes.Use(authMiddleware(s.tokenMaker))
	{
		authenticatedRoutes.POST("/urls", s.CreateUrl)
		authenticatedRoutes.GET("/urls", s.GetListURLsOfUser)
		authenticatedRoutes.GET("/all-urls", s.GetListURLs)

		authenticatedRoutes.GET("/profile", s.getProfile)
	}

	userRoutes := r.Group("/users")
	userRoutes.Use(authMiddleware(s.tokenMaker))
	{
		userRoutes.GET("/profile", s.getProfile)
	}

	s.router = r
}

func (s *Server) Start(address string) error {
	return s.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

func returnGinError(ctx *gin.Context, httpErrorCode int, err interface{}) {
	ctx.JSON(httpErrorCode, err)
}
