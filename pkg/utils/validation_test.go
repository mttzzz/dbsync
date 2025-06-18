package utils

import (
	"testing"
)

func TestValidateFileName(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "valid filename",
			filename: "test_file.sql",
			wantErr:  false,
		},
		{
			name:     "valid filename with numbers",
			filename: "backup_2023_12_25.sql",
			wantErr:  false,
		},
		{
			name:     "empty filename",
			filename: "",
			wantErr:  true,
		},
		{
			name:     "filename with invalid characters",
			filename: "test<>file.sql",
			wantErr:  true,
		},
		{
			name:     "filename too long",
			filename: "this_is_a_very_long_filename_that_exceeds_the_maximum_allowed_length_for_most_filesystems_and_should_be_rejected_by_validation_function_because_it_is_way_too_long_to_be_practical_and_goes_well_beyond_the_255_character_limit_that_we_have_set_and_this_additional_text_makes_it_even_longer.sql",
			wantErr:  true,
		},
		{
			name:     "filename with path",
			filename: "../../../etc/passwd",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileName(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFileName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal input",
			input:    "test_database",
			expected: "test_database",
		},
		{
			name:     "input with spaces",
			input:    "  test database  ",
			expected: "test database",
		},
		{
			name:     "input with special characters",
			input:    "test@database#name",
			expected: "test@database#name",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "input with newlines",
			input:    "test\ndatabase\nname",
			expected: "test database name",
		},
		{
			name:     "input with tabs",
			input:    "test\tdatabase\tname",
			expected: "test database name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeInput(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeInput() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "bytes",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "kilobytes",
			bytes:    1536, // 1.5 KB
			expected: "1.5 KB",
		},
		{
			name:     "megabytes",
			bytes:    2097152, // 2 MB
			expected: "2.0 MB",
		},
		{
			name:     "gigabytes",
			bytes:    3221225472, // 3 GB
			expected: "3.0 GB",
		},
		{
			name:     "terabytes",
			bytes:    1099511627776, // 1 TB
			expected: "1.0 TB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBytes(tt.bytes)
			if got != tt.expected {
				t.Errorf("FormatBytes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsDangerous(t *testing.T) {
	tests := []struct {
		name     string
		dbName   string
		expected bool
	}{
		{
			name:     "production database",
			dbName:   "production",
			expected: true,
		},
		{
			name:     "prod database",
			dbName:   "prod",
			expected: true,
		},
		{
			name:     "live database",
			dbName:   "live",
			expected: true,
		},
		{
			name:     "master database",
			dbName:   "master",
			expected: true,
		},
		{
			name:     "main database",
			dbName:   "main",
			expected: true,
		},
		{
			name:     "test database",
			dbName:   "test",
			expected: false,
		},
		{
			name:     "development database",
			dbName:   "development",
			expected: false,
		},
		{
			name:     "staging database",
			dbName:   "staging",
			expected: false,
		},
		{
			name:     "custom database",
			dbName:   "my_custom_db",
			expected: false,
		},
		{
			name:     "case insensitive production",
			dbName:   "PRODUCTION",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDangerous(tt.dbName)
			if got != tt.expected {
				t.Errorf("IsDangerous() = %v, want %v", got, tt.expected)
			}
		})
	}
}
