// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
)

func getHookTemplates() (hookNames, hookTpls, giteaHookTpls []string) {
	hookNames = []string{"pre-receive", "update", "post-receive"}
	hookTpls = []string{
		// for pre-receive
		fmt.Sprintf(`#!/usr/bin/env %s
# AUTO GENERATED BY GITEA, DO NOT MODIFY
data=$(cat)
exitcodes=""
hookname=$(basename $0)
GIT_DIR=${GIT_DIR:-$(dirname $0)/..}

for hook in ${GIT_DIR}/hooks/${hookname}.d/*; do
  test -x "${hook}" && test -f "${hook}" || continue
  echo "${data}" | "${hook}"
  exitcodes="${exitcodes} $?"
done

for i in ${exitcodes}; do
  [ ${i} -eq 0 ] || exit ${i}
done
`, setting.ScriptType),

		// for update
		fmt.Sprintf(`#!/usr/bin/env %s
# AUTO GENERATED BY GITEA, DO NOT MODIFY
exitcodes=""
hookname=$(basename $0)
GIT_DIR=${GIT_DIR:-$(dirname $0/..)}

for hook in ${GIT_DIR}/hooks/${hookname}.d/*; do
  test -x "${hook}" && test -f "${hook}" || continue
  "${hook}" $1 $2 $3
  exitcodes="${exitcodes} $?"
done

for i in ${exitcodes}; do
  [ ${i} -eq 0 ] || exit ${i}
done
`, setting.ScriptType),

		// for post-receive
		fmt.Sprintf(`#!/usr/bin/env %s
# AUTO GENERATED BY GITEA, DO NOT MODIFY
data=$(cat)
exitcodes=""
hookname=$(basename $0)
GIT_DIR=${GIT_DIR:-$(dirname $0)/..}

for hook in ${GIT_DIR}/hooks/${hookname}.d/*; do
  test -x "${hook}" && test -f "${hook}" || continue
  echo "${data}" | "${hook}"
  exitcodes="${exitcodes} $?"
done

for i in ${exitcodes}; do
  [ ${i} -eq 0 ] || exit ${i}
done
`, setting.ScriptType),
	}

	giteaHookTpls = []string{
		// for pre-receive
		fmt.Sprintf(`#!/usr/bin/env %s
# AUTO GENERATED BY GITEA, DO NOT MODIFY
%s hook --config=%s pre-receive
`, setting.ScriptType, util.ShellEscape(setting.AppPath), util.ShellEscape(setting.CustomConf)),

		// for update
		fmt.Sprintf(`#!/usr/bin/env %s
# AUTO GENERATED BY GITEA, DO NOT MODIFY
%s hook --config=%s update $1 $2 $3
`, setting.ScriptType, util.ShellEscape(setting.AppPath), util.ShellEscape(setting.CustomConf)),

		// for post-receive
		fmt.Sprintf(`#!/usr/bin/env %s
# AUTO GENERATED BY GITEA, DO NOT MODIFY
%s hook --config=%s post-receive
`, setting.ScriptType, util.ShellEscape(setting.AppPath), util.ShellEscape(setting.CustomConf)),
	}

	if git.SupportProcReceive {
		hookNames = append(hookNames, "proc-receive")
		hookTpls = append(hookTpls,
			fmt.Sprintf(`#!/usr/bin/env %s
# AUTO GENERATED BY GITEA, DO NOT MODIFY
%s hook --config=%s proc-receive
`, setting.ScriptType, util.ShellEscape(setting.AppPath), util.ShellEscape(setting.CustomConf)))
		giteaHookTpls = append(giteaHookTpls, "")
	}

	return
}

// CreateDelegateHooks creates all the hooks scripts for the repo
func CreateDelegateHooks(repoPath string) error {
	return createDelegateHooks(repoPath)
}

// createDelegateHooks creates all the hooks scripts for the repo
func createDelegateHooks(repoPath string) (err error) {
	hookNames, hookTpls, giteaHookTpls := getHookTemplates()
	hookDir := filepath.Join(repoPath, "hooks")

	for i, hookName := range hookNames {
		oldHookPath := filepath.Join(hookDir, hookName)
		newHookPath := filepath.Join(hookDir, hookName+".d", "gitea")

		if err := os.MkdirAll(filepath.Join(hookDir, hookName+".d"), os.ModePerm); err != nil {
			return fmt.Errorf("create hooks dir '%s': %v", filepath.Join(hookDir, hookName+".d"), err)
		}

		// WARNING: This will override all old server-side hooks
		if err = util.Remove(oldHookPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("unable to pre-remove old hook file '%s' prior to rewriting: %v ", oldHookPath, err)
		}
		if err = os.WriteFile(oldHookPath, []byte(hookTpls[i]), 0o777); err != nil {
			return fmt.Errorf("write old hook file '%s': %v", oldHookPath, err)
		}

		if err = ensureExecutable(oldHookPath); err != nil {
			return fmt.Errorf("Unable to set %s executable. Error %v", oldHookPath, err)
		}

		if err = util.Remove(newHookPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("unable to pre-remove new hook file '%s' prior to rewriting: %v", newHookPath, err)
		}
		if err = os.WriteFile(newHookPath, []byte(giteaHookTpls[i]), 0o777); err != nil {
			return fmt.Errorf("write new hook file '%s': %v", newHookPath, err)
		}

		if err = ensureExecutable(newHookPath); err != nil {
			return fmt.Errorf("Unable to set %s executable. Error %v", oldHookPath, err)
		}
	}

	return nil
}

func checkExecutable(filename string) bool {
	// windows has no concept of a executable bit
	if runtime.GOOS == "windows" {
		return true
	}
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & 0o100) > 0
}

func ensureExecutable(filename string) error {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return err
	}
	if (fileInfo.Mode() & 0o100) > 0 {
		return nil
	}
	mode := fileInfo.Mode() | 0o100
	return os.Chmod(filename, mode)
}

// CheckDelegateHooks checks the hooks scripts for the repo
func CheckDelegateHooks(repoPath string) ([]string, error) {
	hookNames, hookTpls, giteaHookTpls := getHookTemplates()

	hookDir := filepath.Join(repoPath, "hooks")
	results := make([]string, 0, 10)

	for i, hookName := range hookNames {
		oldHookPath := filepath.Join(hookDir, hookName)
		newHookPath := filepath.Join(hookDir, hookName+".d", "gitea")

		cont := false
		isExist, err := util.IsExist(oldHookPath)
		if err != nil {
			results = append(results, fmt.Sprintf("unable to check if %s exists. Error: %v", oldHookPath, err))
		}
		if err == nil && !isExist {
			results = append(results, fmt.Sprintf("old hook file %s does not exist", oldHookPath))
			cont = true
		}
		isExist, err = util.IsExist(oldHookPath + ".d")
		if err != nil {
			results = append(results, fmt.Sprintf("unable to check if %s exists. Error: %v", oldHookPath+".d", err))
		}
		if err == nil && !isExist {
			results = append(results, fmt.Sprintf("hooks directory %s does not exist", oldHookPath+".d"))
			cont = true
		}
		isExist, err = util.IsExist(newHookPath)
		if err != nil {
			results = append(results, fmt.Sprintf("unable to check if %s exists. Error: %v", newHookPath, err))
		}
		if err == nil && !isExist {
			results = append(results, fmt.Sprintf("new hook file %s does not exist", newHookPath))
			cont = true
		}
		if cont {
			continue
		}
		contents, err := os.ReadFile(oldHookPath)
		if err != nil {
			return results, err
		}
		if string(contents) != hookTpls[i] {
			results = append(results, fmt.Sprintf("old hook file %s is out of date", oldHookPath))
		}
		if !checkExecutable(oldHookPath) {
			results = append(results, fmt.Sprintf("old hook file %s is not executable", oldHookPath))
		}
		contents, err = os.ReadFile(newHookPath)
		if err != nil {
			return results, err
		}
		if string(contents) != giteaHookTpls[i] {
			results = append(results, fmt.Sprintf("new hook file %s is out of date", newHookPath))
		}
		if !checkExecutable(newHookPath) {
			results = append(results, fmt.Sprintf("new hook file %s is not executable", newHookPath))
		}
	}
	return results, nil
}
