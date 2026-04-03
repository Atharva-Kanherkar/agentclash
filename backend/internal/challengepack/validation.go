package challengepack

import (
	"fmt"
	"strings"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/scoring"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	parts := make([]string, 0, len(e))
	for _, item := range e {
		parts = append(parts, item.Error())
	}
	return strings.Join(parts, "; ")
}

func ValidateBundle(bundle Bundle) error {
	var errs ValidationErrors

	if bundle.Pack.Slug == "" {
		errs = append(errs, ValidationError{Field: "pack.slug", Message: "is required"})
	}
	if bundle.Pack.Name == "" {
		errs = append(errs, ValidationError{Field: "pack.name", Message: "is required"})
	}
	if bundle.Pack.Family == "" {
		errs = append(errs, ValidationError{Field: "pack.family", Message: "is required"})
	}
	if bundle.Version.Number <= 0 {
		errs = append(errs, ValidationError{Field: "version.number", Message: "must be greater than 0"})
	}
	if len(bundle.Challenges) == 0 {
		errs = append(errs, ValidationError{Field: "challenges", Message: "must contain at least one challenge"})
	}

	challengeKeys := map[string]struct{}{}
	for i, challenge := range bundle.Challenges {
		path := fmt.Sprintf("challenges[%d]", i)
		if challenge.Key == "" {
			errs = append(errs, ValidationError{Field: path + ".key", Message: "is required"})
		} else {
			if _, exists := challengeKeys[challenge.Key]; exists {
				errs = append(errs, ValidationError{Field: path + ".key", Message: "must be unique"})
			}
			challengeKeys[challenge.Key] = struct{}{}
		}
		if challenge.Title == "" {
			errs = append(errs, ValidationError{Field: path + ".title", Message: "is required"})
		}
		if challenge.Category == "" {
			errs = append(errs, ValidationError{Field: path + ".category", Message: "is required"})
		}
		switch challenge.Difficulty {
		case "easy", "medium", "hard", "expert":
		default:
			errs = append(errs, ValidationError{Field: path + ".difficulty", Message: "must be one of easy, medium, hard, expert"})
		}
		errs = append(errs, validateAssets(path+".assets", challenge.Assets)...)
	}

	inputSetKeys := map[string]struct{}{}
	for i, inputSet := range bundle.InputSets {
		path := fmt.Sprintf("input_sets[%d]", i)
		if inputSet.Key == "" {
			errs = append(errs, ValidationError{Field: path + ".key", Message: "is required"})
		} else {
			if _, exists := inputSetKeys[inputSet.Key]; exists {
				errs = append(errs, ValidationError{Field: path + ".key", Message: "must be unique"})
			}
			inputSetKeys[inputSet.Key] = struct{}{}
		}
		if inputSet.Name == "" {
			errs = append(errs, ValidationError{Field: path + ".name", Message: "is required"})
		}
		if len(inputSet.Items) == 0 {
			errs = append(errs, ValidationError{Field: path + ".items", Message: "must contain at least one item"})
		}

		itemKeys := map[string]struct{}{}
		for itemIndex, item := range inputSet.Items {
			itemPath := fmt.Sprintf("%s.items[%d]", path, itemIndex)
			if item.ChallengeKey == "" {
				errs = append(errs, ValidationError{Field: itemPath + ".challenge_key", Message: "is required"})
			} else {
				if _, exists := challengeKeys[item.ChallengeKey]; !exists {
					errs = append(errs, ValidationError{Field: itemPath + ".challenge_key", Message: "must reference a declared challenge"})
				}
			}
			if item.ItemKey == "" {
				errs = append(errs, ValidationError{Field: itemPath + ".item_key", Message: "is required"})
			} else {
				composite := item.ChallengeKey + "\x00" + item.ItemKey
				if _, exists := itemKeys[composite]; exists {
					errs = append(errs, ValidationError{Field: itemPath + ".item_key", Message: "must be unique per challenge in the input set"})
				}
				itemKeys[composite] = struct{}{}
			}
			errs = append(errs, validateAssets(itemPath+".assets", item.Assets)...)
		}
	}

	if err := scoring.ValidateEvaluationSpec(bundle.Version.EvaluationSpec); err != nil {
		if scoringErrs, ok := err.(scoring.ValidationErrors); ok {
			for _, item := range scoringErrs {
				errs = append(errs, ValidationError{Field: "version." + item.Field, Message: item.Message})
			}
		} else {
			errs = append(errs, ValidationError{Field: "version.evaluation_spec", Message: err.Error()})
		}
	}
	errs = append(errs, validateAssets("version.assets", bundle.Version.Assets)...)

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateAssets(path string, assets []AssetReference) ValidationErrors {
	var errs ValidationErrors
	seen := map[string]struct{}{}
	for i, asset := range assets {
		assetPath := fmt.Sprintf("%s[%d]", path, i)
		if asset.Key == "" {
			errs = append(errs, ValidationError{Field: assetPath + ".key", Message: "is required"})
		} else {
			if _, exists := seen[asset.Key]; exists {
				errs = append(errs, ValidationError{Field: assetPath + ".key", Message: "must be unique"})
			}
			seen[asset.Key] = struct{}{}
		}
		if asset.Path == "" {
			errs = append(errs, ValidationError{Field: assetPath + ".path", Message: "is required"})
		}
	}
	return errs
}
