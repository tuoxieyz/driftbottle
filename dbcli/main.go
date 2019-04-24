package main

import (
	// 	"bufio"
	// 	"errors"

	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	me      *user
	rootCmd = &cobra.Command{
		Use:   "driftbottlecli",
		Short: "driftbottle for chat",
		Long:  "driftbottle is a cli tool for people to chat securely with each other",
	}
	throwCmd = &cobra.Command{
		Use:   "throw",
		Short: "throw a bottle",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("you should put one message in this bottle")
			}
			content := args[0]
			return me.throwBottle(content)
		},
	}
	getMyBottlesCmd = &cobra.Command{
		Use:   "mybottles",
		Short: "get all bottles I thrown",
		RunE: func(cmd *cobra.Command, args []string) error {
			return me.getMyBottles()
		},
	}
	getBottleCmd = &cobra.Command{
		Use:   "bottle",
		Short: "get a bottle",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("miss bottle uuid")
			}
			bid := args[0]
			me.getBottle(bid)
			return nil
		},
	}
	salvageCmd = &cobra.Command{
		Use:   "salvage",
		Short: "salvage a bottle",
		Run: func(cmd *cobra.Command, args []string) {
			me.salvage()
		},
	}
	getMessageOfBottleCmd = &cobra.Command{
		Use:   "msgofbottle",
		Short: "get messages in a bottle",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("miss some arguments")
			}
			bid := args[0]
			mid, _ := cmd.Flags().GetUint16("mid")
			me.getMessageOfBottle(bid, mid)
		},
	}
	replyCmd = &cobra.Command{
		Use:   "reply",
		Short: "reply to otherside",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("miss some arguments")
			}
			bid := args[0]
			content, _ := cmd.Flags().GetString("content")
			me.reply(bid, content)
		},
	}
)

func init() {
	user, err := loadOrGenUserKey()
	if err != nil {
		panic(err)
	}
	me = user

	//rootCmd.PersistentFlags().String("nodekeyfile", "", "设置节点私钥存储路径")
	rootCmd.AddCommand(throwCmd)
	rootCmd.AddCommand(getMyBottlesCmd)
	rootCmd.AddCommand(getBottleCmd)
	rootCmd.AddCommand(salvageCmd)

	getMessageOfBottleCmd.Flags().Uint16("mid", 0, "要获取瓶中的第几条消息")
	rootCmd.AddCommand(getMessageOfBottleCmd)

	replyCmd.Flags().StringP("content", "c", "", "给瓶子的那一头回个消息")
	rootCmd.AddCommand(replyCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
