package runtime

import (
	"math/rand"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "msl2go",
	Short: "Runtime for translated msl script",
}

func AddEvent(name string, f func()) {
	cmd := &cobra.Command{
		Use:   name,
		Short: "event for " + name,
		Run:   func(cmd *cobra.Command, args []string) { f() },
	}
	cmd.Flags().StringVarP(&mslEv.Level, "level", "", "*", "User level")
	cmd.Flags().StringVarP(&mslEv.Location, "location", "", "*", "Event location")
	cmd.Flags().StringVarP(&mslEv.Nick, "nick", "", os.Getenv("SITBOT_FROM"), "Triggering nick")
	cmd.Flags().StringVarP(&mslEv.Chan, "chan", "", os.Getenv("SITBOT_CHAN"), "Triggering channel")
	cmd.Flags().StringVarP(&mslEv.Msg, "msg", "", os.Getenv("SITBOT_MSG"), "Triggering message")
	rootCmd.AddCommand(cmd)
}

func Start() {
	rand.Seed(time.Now().UnixNano())
	mslVar.Load("mslctx.gob")
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
	mslVar.Save("mslctx.gob")
}
