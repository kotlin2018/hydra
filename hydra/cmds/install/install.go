package install

import (
	"os"

	"github.com/micro-plat/cli/cmds"
	"github.com/micro-plat/cli/logs"
	"github.com/micro-plat/hydra/global/compatible"
	"github.com/micro-plat/hydra/global"
	hydracmds "github.com/micro-plat/hydra/hydra/cmds"
	"github.com/micro-plat/lib4go/errs"
	"github.com/urfave/cli"
)

func init() {
	cmds.RegisterFunc(func() cli.Command {
		return cli.Command{
			Name:   "install",
			Usage:  "安装本地服务。安装完成后可通过'start'命令启动服务",
			Flags:  getFlags(),
			Action: doInstall,
		}
	})
}

func doInstall(c *cli.Context) (err error) {

	//1.检查是否有管理员权限
	global.Current().Log().Pause()
	if err = compatible.CheckPrivileges(); err != nil {
		return err
	}

	//2. 绑定应用程序参数
	if err := global.Def.Bind(c); err != nil {
		logs.Log.Error(err)
		cli.ShowCommandHelp(c, c.Command.Name)
		return nil
	}
	args := []string{"run"}
	args = append(args, os.Args[2:]...)
	//3.创建本地服务
	hydraSrv, err := hydracmds.GetService(c, args...)
	if err != nil {
		return err
	}
	if coverIfExists {
		hydraSrv.Uninstall()
	}

	err = hydraSrv.Install()
	if err != nil {
		return err
	}
	return errs.NewIgnoreError(0, err)
}
