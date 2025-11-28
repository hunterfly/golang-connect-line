package protocal

import (
	"flag"
	"golang-template/configs"
	httpAdapter "golang-template/internal/adapters/input/http"
	lineAdapter "golang-template/internal/adapters/output/line"
	"golang-template/internal/adapters/output/postgres"
	"golang-template/internal/application"
	"golang-template/pkg/database_driver/gorm"
	"log"
	"os"
	"os/signal"

	swagger "github.com/arsmn/fiber-swagger/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/sirupsen/logrus"
)

type config struct {
	ENV string `mapstructure:"env"`
}

// ServeHTTP func
func ServeHTTP() error {
	app := fiber.New()
	var cfg config
	flag.StringVar(&cfg.ENV, "env", "", "the environment to use")
	flag.Parse()
	configs.InitViper("./configs", cfg.ENV)
	logrus.Info(configs.GetViper().Env)
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept,Authorization",
	}))
	dbConGorm, err := gorm.ConnectToPostgreSQL(
		configs.GetViper().Postgres.Host,
		configs.GetViper().Postgres.Port,
		configs.GetViper().Postgres.Username,
		configs.GetViper().Postgres.Password,
		configs.GetViper().Postgres.DbName,
		configs.GetViper().Postgres.SSLMode,
	)
	if err != nil {
		return err
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Println("Gracefull shut down ...")
			gorm.DisconnectPostgres(dbConGorm.Postgres)
			err := app.Shutdown()
			if err != nil {
				log.Println("Error when shutdown server: ", err)
			}
		}
	}()

	// Wire up the hexagonal architecture layers
	// Output adapter (repository)
	postgresRepo := postgres.NewTodoRepository(dbConGorm.Postgres)
	// Application service (use case)
	srv := application.NewTodoService(postgresRepo)
	// Input adapter (HTTP handler)
	hdl := httpAdapter.New(srv, dbConGorm.Postgres)

	// Wire up LINE hexagonal architecture
	// Output adapter (LINE client)
	lineClient, err := lineAdapter.NewLineClientAdapter(configs.GetViper().Line.ChannelToken)
	if err != nil {
		logrus.Fatalf("Failed to create LINE client: %v", err)
	}
	// Application service (LINE webhook use case)
	lineWebhookSrv := application.NewLineWebhookService(lineClient, postgresRepo)
	// Input adapter (LINE webhook handler)
	lineWebhookHdl := httpAdapter.NewLineWebhookHandler(lineWebhookSrv, configs.GetViper().Line.ChannelSecret)
	app.Get("/swagger/*", swagger.HandlerDefault) // default
	app.Get("/health", hdl.HealthCheck)

	magnolia := app.Group("/v1/api")
	{
		magnolia.Post("/todo", hdl.CreateTodo)
		magnolia.Put("/todo", hdl.UpdateTodo)
		magnolia.Delete("/todo/:id", hdl.DeleteTodo)
		magnolia.Get("/todo/:id", hdl.GetTodo)
		magnolia.Get("/todo", hdl.GetTodo)
	}

	// LINE webhook endpoint
	webhook := app.Group("/webhook")
	{
		webhook.Post("/line", lineWebhookHdl.HandleWebhook)
	}

	err = app.Listen(":" + configs.GetViper().App.Port)
	if err != nil {
		return err
	}

	logrus.Println("Listerning on port: ", configs.GetViper().App.Port)
	return nil
}
