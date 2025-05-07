package main

type Config struct {
	Bootfile      string `json:"bootfile"`
	TidalFilesDir string `json:"tidal_files_dir"`
	SamplesDir    string `json:"samples_dir"`
}
