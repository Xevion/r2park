package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func RegisterModalHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	// TODO: Pull in all parameters from the form
	// TOOD: Pull in all hidden parameters form database
	// TODO: Pull in resident ID from database

	err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Registration data received. Please wait while your vehicle is registered.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		panic(err)
	}

	data := interaction.ModalSubmitData()
	dataByCustomID := lo.SliceToMap(data.Components, func(c discordgo.MessageComponent) (string, discordgo.MessageComponent) {
		inner := c.(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput)
		return inner.CustomID, inner
	})

	email := dataByCustomID["email"].(*discordgo.TextInput).Value
	log.Infof("Email: %s", email)

	// TOOD: Submit registration request to API
	// TODO: Edit response to indicate success/failure
	// TOOD: On success, provide a button to submit email confirmation

	// TODO: Validate license plate
	// session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
	// 	Type: discordgo.InteractionResponseChannelMessageWithSource,
	// 	Data: &discordgo.InteractionResponseData{
	// 		Embeds: []*discordgo.MessageEmbed{
	// 			{
	// 				Footer: &discordgo.MessageEmbedFooter{
	// 					Text: GetFooterText(),
	// 				},
	// 				Description: "testing 123",
	// 				Fields:      []*discordgo.MessageEmbedField{},
	// 			},
	// 		},
	// 		AllowedMentions: &discordgo.MessageAllowedMentions{},
	// 	},
	// })
}
