package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	session            *discordgo.Session
	commandDefinitions = []*discordgo.ApplicationCommand{RegisterCommandDefinition}
	commandHandlers    = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"register": RegisterCommandHandler,
	}
)

var RegisterCommandDefinition = &discordgo.ApplicationCommand{
	Name:        "register",
	Description: "Register a vehicle for parking",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         "location",
			Description:  "The complex to register with",
			Required:     true,
			Autocomplete: true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "make",
			Description: "Make of Vehicle (e.g. Honda)",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "model",
			Description: "Model of Vehicle (e.g. Civic)",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "plate",
			Description: "License Plate Number (e.g. 123ABC)",
			Required:    true,
		},
	},
}

func RegisterCommandHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	switch interaction.Type {
	case discordgo.InteractionApplicationCommand:
		session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Footer: &discordgo.MessageEmbedFooter{
							Text: fmt.Sprintf("Fetched at %s", time.Now().Format("Monday, January 2, 2006 at 3:04:05PM")),
						},
						Description: "",
						Fields:      []*discordgo.MessageEmbedField{},
					},
				},
				AllowedMentions: &discordgo.MessageAllowedMentions{},
			},
		})
	case discordgo.InteractionApplicationCommandAutocomplete:
		data := interaction.ApplicationCommandData()

		var choices []*discordgo.ApplicationCommandOptionChoice

		if data.Options[0].Focused {
			locations := GetLocations()
			log.Printf("%d location%s provided", len(locations), Plural(len(locations)))

			choices = make([]*discordgo.ApplicationCommandOptionChoice, min(len(locations), 25))
			for i, location := range locations {
				choices[i] = &discordgo.ApplicationCommandOptionChoice{
					Name:  location.name,
					Value: strconv.Itoa(int(location.id)),
				}
			}
		} else {
			choices = []*discordgo.ApplicationCommandOptionChoice{}
		}

		err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: choices, // This is basically the whole purpose of autocomplete interaction - return custom options to the user.
			},
		})
		if err != nil {
			panic(err)
		}
	}
}

// func test() {
// 	body := EncodeForm(
// 		map[string]string{
// 			"propertyIdSelected": "22167",
// 			"propertySource":     "parking-snap",
// 			"guestCode":          code,
// 		})

// 	req, err := http.NewRequest("POST", url, strings.NewReader(body))
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
// 	req.Header.Set("Referer", "https://www.register2park.com/register?key=678zv9zzylvw")
// 	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0")
// 	req.Header.Set("X-Requested-With", "XMLHttpRequest")
// 	req.Header.Set("Host", "www.register2park.com")
// 	req.Header.Set("Cookie", "PHPSESSID=dbc956tgnapqv6l81ue56uf1ng")

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	if resp.StatusCode == 200 {
// 	}

// }

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	session, err := discordgo.New("Bot " + os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)

		// Count serers
		guilds := s.State.Guilds
		log.Printf("Connected to %d server%s", len(guilds), Plural(len(guilds)))
	})
	err = session.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	session.AddHandler(func(internalSession *discordgo.Session, interaction *discordgo.InteractionCreate) {
		if handler, ok := commandHandlers[interaction.ApplicationCommandData().Name]; ok {
			handler(internalSession, interaction)
		}
	})

	log.Printf("Adding %d command%s...", len(commandDefinitions), Plural(len(commandDefinitions)))
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commandDefinitions))
	for definitionIndex, commandDefinition := range commandDefinitions {
		command, err := session.ApplicationCommandCreate(session.State.User.ID, os.Getenv("BOT_TARGET_GUILD"), commandDefinition)
		if err != nil {
			log.Panicf("Failed while registering '%v' command: %v", commandDefinition.Name, err)
		}
		registeredCommands[definitionIndex] = command
	}

	defer session.Close()

	tryReload("")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	if true {
		log.Printf("Removing %d command%s...\n", len(registeredCommands), Plural(len(registeredCommands)))

		for _, v := range registeredCommands {
			err := session.ApplicationCommandDelete(session.State.User.ID, os.Getenv("BOT_TARGET_GUILD"), v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	log.Println("Gracefully shutting down.")
}
