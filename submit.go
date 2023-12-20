package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
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
	spew.Dump(data)

	// TOOD: Submit registration request to API
	// TODO: Edit response to indicate success/failure
	// TOOD: On success, provide a button to submit email confirmation
}
