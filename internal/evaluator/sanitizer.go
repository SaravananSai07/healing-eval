package evaluator

import (
	"strings"

	"github.com/saisaravanan/healing-eval/internal/domain"
)

type MessageSanitizer struct {
	maxMessageLength int
	maxTotalLength   int
}

func NewMessageSanitizer() *MessageSanitizer {
	return &MessageSanitizer{
		maxMessageLength: 4000,  // Max chars per message
		maxTotalLength:   15000, // Max chars for all messages
	}
}

// TruncateMessage safely truncates a single message
func (s *MessageSanitizer) TruncateMessage(content string) string {
	if len(content) <= s.maxMessageLength {
		return content
	}

	// Keep first 60% and last 40%
	keepStart := int(float64(s.maxMessageLength) * 0.6)
	keepEnd := s.maxMessageLength - keepStart - 50 // 50 chars for ellipsis marker

	return content[:keepStart] +
		"\n\n[... content truncated for length ...]\n\n" +
		content[len(content)-keepEnd:]
}

// SanitizeForEvaluation prevents prompt injection attacks
func (s *MessageSanitizer) SanitizeForEvaluation(content string) string {
	// Remove common prompt injection patterns
	injectionPatterns := []string{
		"ignore previous instructions",
		"ignore all previous",
		"disregard previous",
		"forget everything",
		"new instructions:",
		"system:",
		"assistant:",
		"[SYSTEM]",
		"[INST]",
		"</s>",
		"<|im_end|>",
		"<|endoftext|>",
		"<|im_start|>",
		"[/INST]",
		"<system>",
		"</system>",
		"<assistant>",
		"</assistant>",
		"jailbreak",
		"pretend you are",
		"act as",
		"roleplay as",
	}

	sanitized := strings.ToLower(content)
	for _, pattern := range injectionPatterns {
		if strings.Contains(sanitized, pattern) {
			// Replace with safe marker (case-insensitive)
			lowerPattern := strings.ToLower(pattern)
			for i := 0; i < len(content); {
				if i+len(pattern) > len(content) {
					break
				}
				if strings.ToLower(content[i:i+len(pattern)]) == lowerPattern {
					content = content[:i] + "[SANITIZED]" + content[i+len(pattern):]
					i += len("[SANITIZED]")
				} else {
					i++
				}
			}
		}
	}

	return content
}

// PrepareConversationForEval applies all protections
func (s *MessageSanitizer) PrepareConversationForEval(turns []domain.Turn) []domain.Turn {
	sanitized := make([]domain.Turn, 0, len(turns))
	totalLength := 0

	for i, turn := range turns {
		// Create a copy to avoid modifying original
		newTurn := turn

		// Sanitize for prompt injection
		content := s.SanitizeForEvaluation(newTurn.Content)

		// Truncate if needed
		content = s.TruncateMessage(content)

		// Check total budget
		totalLength += len(content)
		if totalLength > s.maxTotalLength {
			// Stop including more turns
			return sanitized
		}

		newTurn.Content = content
		sanitized = append(sanitized, newTurn)

		// Prevent infinite loops - if we've processed all turns, break
		if i >= len(turns)-1 {
			break
		}
	}

	return sanitized
}
