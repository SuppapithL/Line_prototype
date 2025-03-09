'use client';

import { useState } from 'react';

export default function LibraryPage() {
  // All available themes (could be fetched from an API)
  const availableThemes = ['Theme 1', 'Theme 2', 'Theme 3', 'Theme 4'];

  // State for selected themes and the input box value
  const [selectedThemes, setSelectedThemes] = useState<string[]>([]);
  const [inputValue, setInputValue] = useState('');

  // Sample files: each file can have multiple themes
  const files = [
    { name: 'File 1', themes: ['Theme 1', 'Theme 2'] },
    { name: 'File 2', themes: ['Theme 2', 'Theme 3'] },
    { name: 'File 3', themes: ['Theme 1', 'Theme 3'] },
    { name: 'File 4', themes: ['Theme 4'] },
    { name: 'File 5', themes: ['Theme 2', 'Theme 4'] },
  ];

  // Toggle a theme: add if not selected, remove if selected.
  const toggleTheme = (theme: string) => {
    setSelectedThemes((prev) =>
      prev.includes(theme) ? prev.filter((t) => t !== theme) : [...prev, theme]
    );
  };

  // Filter files: if no themes are selected, show all files.
  // Otherwise, show only files that include ALL selected themes.
  const filteredFiles =
    selectedThemes.length === 0
      ? files
      : files.filter((file) =>
          selectedThemes.every((theme) => file.themes.includes(theme))
        );

  // Handle input box changes
  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setInputValue(e.target.value);
  };

  // On Enter, if the input matches an available theme (case-insensitive), toggle it on.
  const handleInputKeyUp = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      const match = availableThemes.find(
        (t) => t.toLowerCase() === inputValue.trim().toLowerCase()
      );
      if (match) {
        toggleTheme(match);
      }
      setInputValue('');
    }
  };

  // Suggestions: available themes that contain the input (and are not already selected)
  const suggestions = availableThemes.filter(
    (theme) =>
      theme.toLowerCase().includes(inputValue.trim().toLowerCase()) &&
      !selectedThemes.includes(theme)
  );

  return (
    <div style={{ position: 'relative', minHeight: '100vh', paddingBottom: '140px' }}>
      {/* Top Box: Display selected themes */}
      <div style={{ padding: '10px', borderBottom: '1px solid #ddd' }}>
        {selectedThemes.length > 0 && (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '10px' }}>
            {selectedThemes.map((theme) => (
              <div
                key={theme}
                onClick={() => toggleTheme(theme)}
                style={{
                  padding: '8px 16px',
                  backgroundColor: '#007bff',
                  color: '#fff',
                  borderRadius: '16px',
                  cursor: 'pointer',
                  transition: 'background-color 0.3s ease',
                }}
              >
                {theme}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* File Display Area */}
      <div style={{ padding: '10px' }}>
        <h3>Files</h3>
        <ul>
          {filteredFiles.length > 0 ? (
            filteredFiles.map((file, index) => <li key={index}>{file.name}</li>)
          ) : (
            <p>No files match the selected themes.</p>
          )}
        </ul>
      </div>

      {/* Fixed Input Box at the Bottom */}
      <div
        style={{
          position: 'fixed',
          bottom: '60px', // Adjust if your tab navigation is taller
          left: 0,
          right: 0,
          padding: '10px',
          backgroundColor: '#fff',
          borderTop: '1px solid #ddd',
          zIndex: 1000,
        }}
      >
        <div style={{ position: 'relative' }}>
          <input
            type="text"
            placeholder="Type a theme and press Enter..."
            value={inputValue}
            onChange={handleInputChange}
            onKeyUp={handleInputKeyUp}
            style={{
              width: '100%',
              padding: '10px',
              borderRadius: '4px',
              border: '1px solid #ccc',
            }}
          />
          {/* Dropdown suggestions appear above the input */}
          {inputValue && suggestions.length > 0 && (
            <div
              style={{
                position: 'absolute',
                bottom: '100%', // Display above the input
                left: 0,
                right: 0,
                marginBottom: '5px',
                backgroundColor: '#f4f4f4',
                border: '1px solid #ccc',
                borderRadius: '4px',
                maxHeight: '150px',
                overflowY: 'auto',
                zIndex: 2000,
              }}
            >
              {suggestions.map((theme, index) => (
                <div
                  key={index}
                  onClick={() => {
                    toggleTheme(theme);
                    setInputValue('');
                  }}
                  style={{
                    padding: '8px 10px',
                    cursor: 'pointer',
                  }}
                >
                  {theme}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
