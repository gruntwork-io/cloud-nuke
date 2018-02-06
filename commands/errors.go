package commands

type InvalidFlagError struct{}

func (e InvalidFlagError) Error() string {
	return "Invalid flag"
}
