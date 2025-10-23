# Bible Quiz Pro (uBible)

## Overview
Go-based Bible quiz application with theme creator and UTF-8 normalization

## Tech Stack
- Backend: Go + Fiber framework
- Frontend: Vanilla JS, HTML/CSS
- Database: SQLite

## Project Structure
- main.go - Entry point
- handlers/ - HTTP handlers (themes.go, auth.go, practice.go)
- static/settings.html - Theme creator UI (UTF-8 normalization critical)
- models/ - Data models
- database/ - DB setup
- verses/ - Bible text files

## Key Files
- static/settings.html - Theme creator with UTF-8 normalization
- handlers/themes.go - Theme backend API
- verseparser/verseparser.go - Parse verse formats

## Known Issues
- Settings.html needs UTF-8 normalization function to prevent mojibake

## Development
Run: go run main.go
Server: localhost:3000

## Git Workflow
git add .
git commit -m "Description"
git push
