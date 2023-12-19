package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
)

func RegisterModalHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
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
}
