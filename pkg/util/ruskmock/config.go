package ruskmock

// Config contains a list of settings that determine the behavior
// of the `Server`.
type Config struct {
	PassScoreValidation           bool
	PassTransactionValidation     bool
	PassStateTransition           bool
	PassStateTransitionValidation bool
}

// DefaultConfig returns the default configuration for the Rusk mock server.
func DefaultConfig() *Config {
	return &Config{
		PassScoreValidation:           true,
		PassTransactionValidation:     true,
		PassStateTransition:           true,
		PassStateTransitionValidation: true,
	}
}
