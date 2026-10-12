package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// Retirement is an input constraint, not just a cleanup pass. A stale payload
// must be refused before cleanup can remove an installed copy and deploy it again.
// Inspect only distribution paths; a same-named user skill is not stale input.
func refuseRetiredDistributionPayloads(root string, targets []string) error {
	manifest, found, err := loadInstallDeprecationManifest(root)
	if err != nil || !found {
		return err
	}
	for _, target := range targets {
		output := nativeBuildTargetOutputDir(root, target)
		for _, retired := range manifest.RetiredSkills {
			if !validManagedSkillName(retired.Skill) {
				return fmt.Errorf("invalid retired skill name %q", retired.Skill)
			}
			for _, path := range []string{
				filepath.Join(output, "skills", retired.Skill),
				filepath.Join(output, "commands", retired.Skill+".md"),
			} {
				if _, err := os.Lstat(path); os.IsNotExist(err) {
					continue
				} else if err != nil {
					return fmt.Errorf("inspect retired skill payload %s: %w", path, err)
				}
				return fmt.Errorf("stale distribution contains retired skill %q at %s; run loaf build from current sources or reinstall the current Loaf distribution before retrying", retired.Skill, path)
			}
		}
	}
	return nil
}

func refuseRetiredTargetPayloads(options targetInstallOptions) error {
	root := filepath.Dir(filepath.Dir(options.DistDir))
	return refuseRetiredDistributionPayloads(root, []string{options.Target})
}
