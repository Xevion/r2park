package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"

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
	modalHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"register": RegisterModalHandler,
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
		log.WithField("error", err).Panic("Invalid bot parameters")
	}

	// Login handler
	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.WithFields(
			log.Fields{
				"username":      r.User.Username,
				"discriminator": r.User.Discriminator,
				"id":            r.User.ID,
				"guilds":        len(r.Guilds),
				"session":       r.SessionID,
			}).Info("Logged in successfully")

		// Load the session
		tryReload()
	})

	// Open the session
	err = session.Open()
	if err != nil {
		log.WithField("error", err).Panic("Cannot open the session")
	}

	// Make sure sessions and HTTP clients are closed
	defer session.Close()
	defer client.CloseIdleConnections() // HTTP client

	// Setup command handlers
	session.AddHandler(func(internalSession *discordgo.Session, interaction *discordgo.InteractionCreate) {
		switch interaction.Type {
		case discordgo.InteractionApplicationCommand, discordgo.InteractionApplicationCommandAutocomplete:
			if handler, ok := commandHandlers[interaction.ApplicationCommandData().Name]; ok {
				handler(internalSession, interaction)
			}

		case discordgo.InteractionModalSubmit:
			id := interaction.ModalSubmitData().CustomID
			handlerId := id[:strings.Index(id, ":")]

			if handler, ok := modalHandlers[handlerId]; ok {
				handler(internalSession, interaction)
			}

			// if !strings.HasPrefix(data.CustomID, "modals_survey") {
			// 	return
			// }

			// userid := strings.Split(data.CustomID, "_")[2]
			// _, err = s.ChannelMessageSend(*ResultsChannel, fmt.Sprintf(
			// 	"Feedback received. From <@%s>\n\n**Opinion**:\n%s\n\n**Suggestions**:\n%s",
			// 	userid,
			// 	data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value,
			// 	data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value,
			// ))
			// if err != nil {
			// 	panic(err)
			// }
		default:
			log.WithFields(
				log.Fields{
					"type": interaction.Type,
					"ref":  interaction.Message.Reference(),
				}).Warn("Unhandled interaction type")
		}
	})

	// Setup signals before adding commands in case we CTRL+C before/during command registration
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Register commands
	log.WithField("count", len(commandDefinitions)).Info("Registering commands")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commandDefinitions))
	for definitionIndex, commandDefinition := range commandDefinitions {
		command, err := session.ApplicationCommandCreate(session.State.User.ID, os.Getenv("BOT_TARGET_GUILD"), commandDefinition)
		log.WithField("command", commandDefinition.Name).Debug("Registering command")

		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"command": commandDefinition.Name,
			}).Panic("Failed while registering command")
		}
		registeredCommands[definitionIndex] = command
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Info("Press Ctrl+C to exit")
	<-stop

	// Remove commands
	log.WithField("count", len(registeredCommands)).Info("Removing commands")
	for _, registeredCommand := range registeredCommands {
		log.WithFields(log.Fields{
			"command": registeredCommand.Name,
			"id":      registeredCommand.ID,
		}).Debug("Removing command")
		err := session.ApplicationCommandDelete(session.State.User.ID, os.Getenv("BOT_TARGET_GUILD"), registeredCommand.ID)
		if err != nil {
			log.Panicf("Cannot delete '%v' command: %v", registeredCommand.Name, err)
		}
	}

	log.Warn("Gracefully shutting down")
	defer log.Warn("Graceful shutdown complete")
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.WithField("error", err).Panic("Error loading .env file")
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
	log.WithField("ping", ping_result).Info("Redis connection established")

	command := ""
	args := flag.Args()
	if len(args) > 1 {
		command = args[1]
	}

	log.WithField("command", command).Debug("Starting up")
	switch command {
	case "scan":
		log.Info("Scanning...")
		Scan()
	case "bot":
	default:
		log.Info("Starting bot...")
		Bot()
	}
}
