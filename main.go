package main

import (
	"encoding/json"
	"fmt"
	"github.com/akamensky/argparse"
	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/graniet/operative-framework/api"
	"github.com/graniet/operative-framework/compiler"
	"github.com/graniet/operative-framework/cron"
	"github.com/graniet/operative-framework/engine"
	"github.com/graniet/operative-framework/export"
	"github.com/graniet/operative-framework/session"
	"github.com/graniet/operative-framework/supervisor"
	"github.com/joho/godotenv"
	"io"
	"log"
	"os"
	"os/user"
	"strconv"
	"strings"
)

func main() {
	var sess *session.Session
	var sp *supervisor.Supervisor
	var configJob string
	var opfBaseDirectory string
	var opfExport string

	// Load Configuration File
	configFile := ".env"
	err := godotenv.Load(".env")

	if err != nil {

		// Generate Default .env File
		u, errU := user.Current()
		if errU != nil {
			fmt.Println("Please create a .env file on root path.")
			return
		}
		if _, err := os.Stat(u.HomeDir + "/.opf/.env"); os.IsNotExist(err) {
			if _, err := os.Stat(u.HomeDir + "/.opf/"); os.IsNotExist(err) {
				_ = os.Mkdir(u.HomeDir+"/.opf/", os.ModePerm)
			}
			log.Println("Generating default .env on '" + u.HomeDir + "' directory...")
			path, errGeneration := engine.GenerateEnv(u.HomeDir + "/.opf/.env")
			if errGeneration != nil {
				log.Println(errGeneration.Error())
				return
			}
			err := godotenv.Load(path)
			if err != nil {
				log.Println(err.Error())
				return
			}
		}
		configFile = u.HomeDir + "/.opf/.env"
		configJob = u.HomeDir + "/.opf/cron/"
		opfBaseDirectory = u.HomeDir + "/.opf/"
		opfExport = opfBaseDirectory + "export/"
	}

	// Argument parser
	parser := argparse.NewParser("operative-framework", "digital investigation framework")
	rApi := parser.Flag("a", "api", &argparse.Options{
		Required: false,
		Help:     "Load instantly operative framework restful API",
	})
	rSupervisor := parser.Flag("", "cron", &argparse.Options{
		Required: false,
		Help:     "Running supervised cron job(s).",
	})
	verbose := parser.Flag("v", "verbose", &argparse.Options{
		Required: false,
		Help:     "Do not show modules messages response",
	})
	cli := parser.Flag("n", "no-cli", &argparse.Options{
		Required: false,
		Help:     "Do not run framework cli",
	})
	execute := parser.String("e", "execute", &argparse.Options{
		Required: false,
		Help:     "Execute a single module",
	})
	target := parser.String("t", "target", &argparse.Options{
		Required: false,
		Help:     "Set target to '-e/--execute' argument",
	})
	parameters := parser.String("p", "parameters", &argparse.Options{
		Required: false,
		Help:     "Set parameters to '-e/--execute' argument",
	})
	loadSession := parser.Int("s", "session", &argparse.Options{
		Required: false,
		Help:     "Load specific session id",
	})
	onlyModuleOutput := parser.Flag("", "only-module-output", &argparse.Options{
		Required: false,
		Help:     "Do not print a banner information",
	})

	help := parser.Flag("h", "help", &argparse.Options{
		Required: false,
		Help:     "Print help",
	})

	scripts := parser.String("f", "opf", &argparse.Options{
		Required: false,
		Help:     "Run script before prompt starting",
	})

	quiet := parser.Flag("q", "quiet", &argparse.Options{
		Required: false,
		Help:     "Don't prompt operative shell",
	})

	modules := parser.Flag("l", "list", &argparse.Options{
		Required: false,
		Help:     "List available modules",
	})

	jsonExport := parser.Flag("", "json", &argparse.Options{
		Required: false,
		Help:     "Print result with a JSON format",
	})

	csvExport := parser.Flag("", "csv", &argparse.Options{
		Required: false,
		Help:     "Print result with a CSV format",
	})

	sendTo := parser.String("", "to", &argparse.Options{
		Required: false,
		Help:     "Send response to webservice (require --json)",
	})

	err = parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		return
	}

	// Checking if session as been specified
	if *loadSession > 0 {
		sess = engine.Load(*loadSession)
	} else {
		sess = engine.New()
	}

	sess.PushPrompt()
	sess.Config.Common.ConfigurationFile = configFile
	sess.Config.Common.ConfigurationJobs = configJob
	sess.Config.Common.BaseDirectory = opfBaseDirectory
	sess.Config.Common.ExportDirectory = opfExport
	sess.ParseModuleConfig()
	sess.ParseWebServiceConfig()
	apiRest := api.PushARestFul(sess)

	// Load supervised cron job.
	sp = supervisor.GetNewSupervisor(sess)
	cron.Load(sp)

	if *modules {
		sess.ListModules()
		return
	}

	if *rSupervisor {
		// Reading loaded cron job
		sp.Read()
		return
	}

	if *help {
		fmt.Print(parser.Usage(""))
		return
	}

	if *verbose {
		sess.Stream.Verbose = false
	} else {
		if !*onlyModuleOutput {
			c := color.New(color.FgYellow)
			_, _ = c.Println("OPERATIVE FRAMEWORK - DIGITAL INVESTIGATION FRAMEWORK")
			sess.Stream.WithoutDate("Loading a configuration file '" + configFile + "'")
			sess.Stream.WithoutDate("Loading a cron job configuration '" + sess.Config.Common.ConfigurationJobs + "'")
			sess.Stream.WithoutDate("Loading '" + strconv.Itoa(len(sess.Config.Modules)) + "' module(s) configuration(s)")
		}
	}

	if *execute != "" {
		if *target == "" {
			sess.Stream.Error("'-e/--execute' argument need a target argument '-t/--target'")
			return
		}
		module, err := sess.SearchModule(*execute)
		if err != nil {
			sess.Stream.Error(err.Error())
			return
		}

		target, err := sess.AddTarget(module.GetType(), *target)
		if err != nil {
			sess.Stream.Error(err.Error())
			return
		}
		_, _ = module.SetParameter("TARGET", target)

		if *parameters != "" {

			if !strings.Contains(*parameters, "=") {
				sess.Stream.Error("Please use a correct format. example: limit=50;id=1")
				return
			}

			if strings.Contains(*parameters, ";") {
				lists := strings.Split(*parameters, ";")
				for _, parameter := range lists {
					template := strings.Split(parameter, "=")
					_, err := module.SetParameter(template[0], template[1])
					if err != nil {
						sess.Stream.Error(err.Error())
						return
					}
				}
			} else {
				template := strings.Split(*parameters, "=")
				_, err := module.SetParameter(template[0], template[1])
				if err != nil {
					sess.Stream.Error(err.Error())
					return
				}
			}
		}
		if *csvExport {
			sess.Stream.CSV = true
		}
		module.Start()

		if *jsonExport {
			e := export.ExportNow(sess, module)
			j, err := json.Marshal(e)
			if err == nil {
				if *sendTo != "" {
					webservice, err := sess.GetWebService(*sendTo)
					if err != nil {
						sess.Stream.Error(err.Error())
						return
					}
					opfClient := sess.Client
					opfClient.Header.Add("Content-Type", "application/json")
					opfClient.Data = j
					_, err = opfClient.Perform("POST", webservice.URL)
					if err != nil {
						sess.Stream.Error(err.Error())
						return
					}
				}
				print(string(j))
			}
			return
		}
		return
	}

	if *rApi {
		if *cli {
			sess.Stream.Standard("running operative framework api...")
			sess.Stream.Standard("available at : " + apiRest.Server.Addr)
			sess.Information.SetApi(true)
			apiRest.Start()
		} else {
			sess.Stream.Standard("running operative framework api...")
			go apiRest.Start()
			sess.Stream.Standard("available at : " + apiRest.Server.Addr)
			sess.Information.SetApi(true)
		}
	}

	if *scripts != "" {
		compiler.Run(sess, *scripts)
	}

	if *quiet {
		return
	}

	l, errP := readline.NewEx(sess.Prompt)
	if errP != nil {
		panic(errP)
	}
	defer l.Close()

	// Checking in background available interval
	go sess.WaitInterval()

	// Checking in background available monitor
	go sess.WaitMonitor()

	// Checking interval in background
	go sess.WaitAnalytics()

	// Run Operative Framework Menu
	for {
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		// Get Line With Trim Space
		line = strings.TrimSpace(line)

		// Checking Command
		if line == "api run" {
			sess.Stream.Success("API Rest as been started at http://" + sess.Config.Api.Host + ":" + sess.Config.Api.Port)
			go apiRest.Start()
			sess.Information.SetApi(true)
		} else if line == "api stop" {
			_ = apiRest.Server.Close()
			sess.Information.SetApi(false)
		} else {
			if !engine.CommandBase(line, sess) {
				sess.ParseCommand(line)
			}
		}
	}
}
