package main

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/ngtrdai197/url-shorterner/config"
	"github.com/ngtrdai197/url-shorterner/pkg/api"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func main() {
	// load config
	c := loadConfig()
	// migration runs
	migration(c)

	s := api.NewServer(c)

	log.Info().Msgf("listening and serving HTTP on %s", c.PublicApiAddress)
	if err := s.Start(c.PublicApiAddress); err != nil {
		log.Fatal().Err(err).Msg("cannot create public api")
	}
}

func loadConfig() *config.Config {
	viper.SetConfigFile(".env")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w", err))
	}
	c, err := config.GetConfig(validator.New())
	if err != nil {
		panic(fmt.Errorf("config file invalidate with error: %w", err))
	}
	return c
}

func migration(c *config.Config) {
	m, err := migrate.New(c.MigrationUrl, c.DbSource)
	if err != nil {
		log.Fatal().Msgf("migrate.New error=%v", err)
	}

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Info().Msg("db schema is already up to date")
		} else {
			log.Fatal().Msgf("migration up with error=%v", err)
		}
	}
}
