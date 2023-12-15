package main

import (
	"flag"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

var (
	session            *discordgo.Session
	commandDefinitions = []*discordgo.ApplicationCommand{RegisterCommandDefinition, CodeCommandDefinition}
	commandHandlers    = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"register": RegisterCommandHandler,
		"code":     CodeCommandHandler,
	}
	db        *redis.Client
	debugFlag = flag.Bool("debug", false, "Enable debug logging")
)

func init() {
	flag.Parse()
	if *debugFlag {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func Bot() {
	// Setup the session parameters
	var err error
	session, err = discordgo.New("Bot " + os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	// Login handler
	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)

		// Count serers
		guilds := s.State.Guilds
		log.Printf("Connected to %d server%s", len(guilds), Plural(len(guilds)))
	})

	// Open the session
	err = session.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	// Make sure sessions and HTTP clients are closed
	defer session.Close()
	defer client.CloseIdleConnections() // HTTP client

	// Setup command handlers
	session.AddHandler(func(internalSession *discordgo.Session, interaction *discordgo.InteractionCreate) {
		if handler, ok := commandHandlers[interaction.ApplicationCommandData().Name]; ok {
			handler(internalSession, interaction)
		}
	})

	// Setup signals before adding commands in case we CTRL+C before/during command registration
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Register commands
	log.Printf("Adding %d command%s...", len(commandDefinitions), Plural(len(commandDefinitions)))
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commandDefinitions))
	for definitionIndex, commandDefinition := range commandDefinitions {
		command, err := session.ApplicationCommandCreate(session.State.User.ID, os.Getenv("BOT_TARGET_GUILD"), commandDefinition)
		log.Debugf("Registering '%v' command (%v)", commandDefinition.Name, command.ID)

		if err != nil {
			log.Panicf("Failed while registering '%v' command: %v", commandDefinition.Name, err)
		}
		registeredCommands[definitionIndex] = command
	}

	// Load the session
	tryReload()

	// Wait here until CTRL-C or other term signal is received.
	log.Println("Press Ctrl+C to exit")
	<-stop

	// Remove commands
	log.Infof("Removing %d command%s...\n", len(registeredCommands), Plural(len(registeredCommands)))
	for _, v := range registeredCommands {
		log.Debugf("Removing '%v' command (%v)", v.Name, v.ID)
		err := session.ApplicationCommandDelete(session.State.User.ID, os.Getenv("BOT_TARGET_GUILD"), v.ID)
		if err != nil {
			log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
		}
	}

	log.Warn("Gracefully shutting down.")
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	opt := &redis.Options{
		Addr:     os.Getenv("REDIS_HOST"),
		Password: os.Getenv("REDIS_PASSWORD"),
	}
	db = redis.NewClient(opt)

	ping_result, err := db.Ping().Result()
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Redis connection established (%s)", ping_result)

	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "scan":
		log.Info("Running scan")
		Scan()
	case "bot":
		log.Info("Running bot")
		Bot()
	default:
		log.Info("Running bot (default)")
		Bot()
	}
}
