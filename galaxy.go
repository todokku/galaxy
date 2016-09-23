package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/litl/galaxy/commander"
	gconfig "github.com/litl/galaxy/config"
	"github.com/litl/galaxy/log"
	"github.com/litl/galaxy/runtime"
	"github.com/litl/galaxy/utils"

	"github.com/BurntSushi/toml"
	"github.com/codegangsta/cli"
	"github.com/ryanuber/columnize"
)

var (
	serviceRuntime *runtime.ServiceRuntime
	configStore    *gconfig.Store

	initOnce     sync.Once
	buildVersion string
)

var config struct {
	Host string `toml:"host"`
}

func initStore(c *cli.Context) {
	configStore = gconfig.NewStore(uint64(c.Int("ttl")), utils.GalaxyRedisHost(c))
}

// ensure the registry as a redis host, but only once
func initRuntime(c *cli.Context) {
	serviceRuntime = runtime.NewServiceRuntime(
		configStore,
		"",
		"127.0.0.1",
	)
}

func ensureAppParam(c *cli.Context, command string) string {
	app := c.Args().First()
	if app == "" {
		cli.ShowCommandHelp(c, command)
		log.Fatal("ERROR: app name missing")
	}

	exists, err := appExists(app, utils.GalaxyEnv(c))
	if err != nil {
		log.Fatalf("ERROR: can't deteremine if %s exists: %s", app, err)
	}

	if !exists {
		log.Fatalf("ERROR: %s does not exist. Create it first.", app)
	}

	return app
}

func ensureEnvArg(c *cli.Context) {
	if utils.GalaxyEnv(c) == "" {
		log.Fatal("ERROR: env is required.  Pass --env or set GALAXY_ENV")
	}
}

func ensurePoolArg(c *cli.Context) {
	if utils.GalaxyPool(c) == "" {
		log.Fatal("ERROR: pool is required.  Pass --pool or set GALAXY_POOL")
	}
}

func appExists(app, env string) (bool, error) {
	return configStore.AppExists(app, env)
}

func appList(c *cli.Context) {
	initStore(c)
	err := commander.AppList(configStore, utils.GalaxyEnv(c))
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}

func appCreate(c *cli.Context) {
	ensureEnvArg(c)
	initStore(c)

	app := c.Args().First()
	if app == "" {
		cli.ShowCommandHelp(c, "app:create")
		log.Fatal("ERROR: app name missing")
	}

	err := commander.AppCreate(configStore, app, utils.GalaxyEnv(c))
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}

func appDelete(c *cli.Context) {
	ensureEnvArg(c)
	initStore(c)

	app := ensureAppParam(c, "app:delete")

	err := commander.AppDelete(configStore, app, utils.GalaxyEnv(c))
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}

func appDeploy(c *cli.Context) {
	ensureEnvArg(c)
	initStore(c)
	initRuntime(c)

	app := ensureAppParam(c, "app:deploy")

	version := ""
	if len(c.Args().Tail()) == 1 {
		version = c.Args().Tail()[0]
	}

	if version == "" {
		log.Println("ERROR: version missing")
		cli.ShowCommandHelp(c, "app:deploy")
		return
	}

	err := commander.AppDeploy(configStore, serviceRuntime, app, utils.GalaxyEnv(c), version)
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}

func appRestart(c *cli.Context) {
	initStore(c)

	app := ensureAppParam(c, "app:restart")

	err := commander.AppRestart(configStore, app, utils.GalaxyEnv(c))
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}

func appRun(c *cli.Context) {
	ensureEnvArg(c)
	initStore(c)
	initRuntime(c)

	app := ensureAppParam(c, "app:run")

	if len(c.Args()) < 2 {
		log.Fatalf("ERROR: Missing command to run.")
		return
	}

	err := commander.AppRun(configStore, serviceRuntime, app, utils.GalaxyEnv(c), c.Args()[1:])
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}

func appShell(c *cli.Context) {
	ensureEnvArg(c)
	initStore(c)
	initRuntime(c)

	app := ensureAppParam(c, "app:shell")

	err := commander.AppShell(configStore, serviceRuntime, app,
		utils.GalaxyEnv(c), utils.GalaxyPool(c))
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}

func configList(c *cli.Context) {
	ensureEnvArg(c)
	initStore(c)
	app := ensureAppParam(c, "config")

	err := commander.ConfigList(configStore, app, utils.GalaxyEnv(c))
	if err != nil {
		log.Fatalf("ERROR: Unable to list config: %s.", err)
		return
	}
}

func configSet(c *cli.Context) {
	ensureEnvArg(c)
	initStore(c)
	app := ensureAppParam(c, "config:set")

	args := c.Args().Tail()
	err := commander.ConfigSet(configStore, app, utils.GalaxyEnv(c), args)

	if err != nil {
		log.Fatalf("ERROR: Unable to update config: %s.", err)
		return
	}
}

func configUnset(c *cli.Context) {
	ensureEnvArg(c)
	initStore(c)
	app := ensureAppParam(c, "config:unset")

	err := commander.ConfigUnset(configStore, app, utils.GalaxyEnv(c), c.Args().Tail())
	if err != nil {
		log.Fatalf("ERROR: Unable to unset config: %s.", err)
		return
	}
}

func configGet(c *cli.Context) {
	ensureEnvArg(c)
	initStore(c)
	app := ensureAppParam(c, "config:get")

	err := commander.ConfigGet(configStore, app, utils.GalaxyEnv(c), c.Args().Tail())

	if err != nil {
		log.Fatalf("ERROR: Unable to get config: %s.", err)
		return
	}
}

// Return the path for the config directory, and create it if it doesn't exist
func cfgDir() string {
	homeDir := utils.HomeDir()
	if homeDir == "" {
		log.Fatal("ERROR: Unable to determine current home dir. Set $HOME.")
	}

	configDir := filepath.Join(homeDir, ".galaxy")
	_, err := os.Stat(configDir)
	if err != nil && os.IsNotExist(err) {
		err = os.Mkdir(configDir, 0700)
		if err != nil {
			log.Fatal("ERROR: cannot create config directory:", err)
		}
	}
	return configDir
}

func poolAssign(c *cli.Context) {
	ensureEnvArg(c)
	ensurePoolArg(c)
	initStore(c)

	app := ensureAppParam(c, "pool:assign")

	err := commander.AppAssign(configStore, app, utils.GalaxyEnv(c), utils.GalaxyPool(c))
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}

func poolUnassign(c *cli.Context) {
	ensureEnvArg(c)
	ensurePoolArg(c)
	initStore(c)

	app := c.Args().First()
	if app == "" {
		cli.ShowCommandHelp(c, "pool:assign")
		log.Fatal("ERROR: app name missing")
	}

	err := commander.AppUnassign(configStore, app, utils.GalaxyEnv(c), utils.GalaxyPool(c))
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}

func poolCreate(c *cli.Context) {
	ensureEnvArg(c)
	ensurePoolArg(c)
	initStore(c)
	created, err := configStore.CreatePool(utils.GalaxyPool(c), utils.GalaxyEnv(c))
	if err != nil {
		log.Fatalf("ERROR: Could not create pool: %s", err)
		return
	}

	if created {
		log.Printf("Pool %s created\n", utils.GalaxyPool(c))
	} else {
		log.Printf("Pool %s already exists\n", utils.GalaxyPool(c))
	}
}

func poolUpdate(c *cli.Context) {
	ensureEnvArg(c)
	ensurePoolArg(c)
}

func poolList(c *cli.Context) {
	initStore(c)

	envs := []string{utils.GalaxyEnv(c)}
	if utils.GalaxyEnv(c) == "" {
		var err error
		envs, err = configStore.ListEnvs()
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}
	}

	columns := []string{"ENV | POOL | APPS "}

	for _, env := range envs {
		pools, err := configStore.ListPools(env)
		if err != nil {
			log.Fatalf("ERROR: cannot list pools: %s", err)
			return
		}

		if len(pools) == 0 {
			columns = append(columns, strings.Join([]string{
				env,
				"",
				""}, " | "))
			continue
		}

		for _, pool := range pools {

			assigments, err := configStore.ListAssignments(env, pool)
			if err != nil {
				log.Fatalf("ERROR: cannot list pool assignments: %s", err)
			}

			columns = append(columns, strings.Join([]string{
				env,
				pool,
				strings.Join(assigments, ",")}, " | "))
		}

	}
	output := columnize.SimpleFormat(columns)
	log.Println(output)
}

func poolDelete(c *cli.Context) {
	ensureEnvArg(c)
	ensurePoolArg(c)
	initStore(c)
	empty, err := configStore.DeletePool(utils.GalaxyPool(c), utils.GalaxyEnv(c))
	if err != nil {
		log.Fatalf("ERROR: Could not delete pool: %s", err)
		return
	}

	if empty {
		log.Printf("Pool %s deleted\n", utils.GalaxyPool(c))

	} else {
		log.Printf("Pool %s has apps assigned. Unassign them first.\n", utils.GalaxyPool(c))
	}
}

func loadConfig() {
	configFile := filepath.Join(cfgDir(), "galaxy.toml")

	_, err := os.Stat(configFile)
	if err == nil {
		if _, err := toml.DecodeFile(configFile, &config); err != nil {
			log.Fatalf("ERROR: Unable to logout: %s", err)
			return
		}
	}

}

func pgPsql(c *cli.Context) {
	ensureEnvArg(c)
	initStore(c)
	app := ensureAppParam(c, "pg:psql")

	appCfg, err := configStore.GetApp(app, utils.GalaxyEnv(c))
	if err != nil {
		log.Fatalf("ERROR: Unable to run command: %s.", err)
		return
	}

	database_url := appCfg.Env()["DATABASE_URL"]
	if database_url == "" {
		log.Printf("No DATABASE_URL configured.  Set one with config:set first.")
		return
	}

	if !strings.HasPrefix(database_url, "postgres://") {
		log.Printf("DATABASE_URL is not a postgres database.")
		return
	}

	if c.Bool("ro") {
		dbURL, err := url.Parse(database_url)
		if err != nil {
			log.Printf("Invalid DATABASE_URL: %s", database_url)
			return
		}

		qp, err := url.ParseQuery(dbURL.RawQuery)
		if err != nil {
			log.Printf("Invalid DATABASE_URL: %s", database_url)
			return
		}

		options := qp.Get("options")
		if options != "" {
			options += " "
		}
		options += fmt.Sprintf("-c default_transaction_read_only=true")
		qp.Set("options", options)

		dbURL.RawQuery = strings.Replace(qp.Encode(), "+", "%20", -1)

		database_url = dbURL.String()
	}

	cmd := exec.Command("psql", database_url)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Ignore SIGINT while the process is running
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	defer func() {
		signal.Stop(ch)
		close(ch)
	}()

	go func() {
		for {
			_, ok := <-ch
			if !ok {
				break
			}
		}
	}()

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Printf("Command finished with error: %v\n", err)
	}
}

func main() {

	loadConfig()

	// Don't print date, etc. and print to stdout
	log.DefaultLogger = log.New(os.Stdout, "", log.INFO)
	log.DefaultLogger.SetFlags(0)

	app := cli.NewApp()
	app.Name = "galaxy"
	app.Usage = "galaxy cli"
	app.Version = buildVersion
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "registry", Value: "", Usage: "host:port[,host:port,..]"},
		cli.StringFlag{Name: "env", Value: "", Usage: "environment (dev, test, prod, etc.)"},
		cli.StringFlag{Name: "pool", Value: "", Usage: "pool (web, worker, etc.)"},
	}

	app.Commands = []cli.Command{
		{
			Name:        "app",
			Usage:       "list the apps currently created",
			Action:      appList,
			Description: "app",
		},
		{
			Name:        "app:backup",
			Usage:       "backup app configs to a file or stdout",
			Action:      appBackup,
			Description: "app:backup [app[,app2]]",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "file", Usage: "backup filename"},
			},
		},
		{
			Name:        "app:restore",
			Usage:       "restore an app's config",
			Action:      appRestore,
			Description: "app:restore [app[,app2]]",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "file", Usage: "backup filename"},
				cli.BoolFlag{Name: "force", Usage: "force overwrite of existing config"},
			},
		},
		{
			Name:        "app:create",
			Usage:       "create a new app",
			Action:      appCreate,
			Description: "app:create",
		},
		{
			Name:        "app:delete",
			Usage:       "delete a new app",
			Action:      appDelete,
			Description: "app:delete",
		},
		{
			Name:        "app:deploy",
			Usage:       "deploy a new version of an app",
			Action:      appDeploy,
			Description: "app:deploy <app> <version>",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "force", Usage: "force pulling the image"},
			},
		},
		{
			Name:        "app:restart",
			Usage:       "restart an app",
			Action:      appRestart,
			Description: "app:restart <app>",
		},
		{
			Name:        "app:run",
			Usage:       "run a command in a container",
			Action:      appRun,
			Description: "app:run <app> <command>",
		},
		{
			Name:        "app:shell",
			Usage:       "run a bash shell in a container",
			Action:      appShell,
			Description: "app:shell <app>",
		},
		{
			Name:        "config",
			Usage:       "list the config values for an app",
			Action:      configList,
			Description: "config <app>",
		},
		{
			Name:        "config:set",
			Usage:       "set one or more configuration variables",
			Action:      configSet,
			Description: "config:set <app> KEY=VALUE [KEY=VALUE ...]",
		},
		{
			Name:        "config:unset",
			Usage:       "unset one or more configuration variables",
			Action:      configUnset,
			Description: "config:unset <app> KEY [KEY ...]",
		},
		{
			Name:        "config:get",
			Usage:       "display the config value for an app",
			Action:      configGet,
			Description: "config:get <app> KEY [KEY ...]",
		},
		{
			Name:        "pool",
			Usage:       "list the pools",
			Action:      poolList,
			Description: "pool",
		},
		{
			Name:        "pool:assign",
			Usage:       "assign an app to a pool",
			Action:      poolAssign,
			Description: "pool:assign",
		},
		{
			Name:        "pool:unassign",
			Usage:       "unassign an app from a pool",
			Action:      poolUnassign,
			Description: "pool:unassign",
		},

		{
			Name:        "pool:create",
			Usage:       "create a pool",
			Action:      poolCreate,
			Description: "pool:create",
		},
		{
			Name:        "pool:delete",
			Usage:       "deletes a pool",
			Action:      poolDelete,
			Description: "pool:delete",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "y", Usage: "skip confirmation"},
			},
		},
		{
			Name:        "pg:psql",
			Usage:       "connect to database using psql",
			Action:      pgPsql,
			Description: "pg:psql <app>",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "ro", Usage: "read-only connection"},
			},
		},
	}
	app.Run(os.Args)
}
