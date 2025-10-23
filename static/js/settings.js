// static/js/settings.js - Shared Settings Manager
// Single source of truth for quiz settings across all pages

const SettingsManager = {
    // Default settings
    defaults: {
        timeLimit: 20,
        questionCount: 20,
        selectedThemes: []
    },

    // Load settings from localStorage
    loadSettings() {
        try {
            const settingsData = localStorage.getItem('quizSettings');
            const themesData = localStorage.getItem('selectedThemes');
            
            const settings = settingsData ? JSON.parse(settingsData) : {};
            const themes = themesData ? JSON.parse(themesData) : [];

            return {
                timeLimit: Number(settings.timeLimit) || this.defaults.timeLimit,
                questionCount: Number(settings.questionCount) || this.defaults.questionCount,
                selectedThemes: Array.isArray(themes) ? themes : []
            };
        } catch (e) {
            console.error('Error loading settings:', e);
            return { ...this.defaults };
        }
    },

    // Save settings to localStorage
    saveSettings(timeLimit, questionCount) {
        try {
            const settings = {
                timeLimit: Number(timeLimit),
                questionCount: Number(questionCount)
            };
            localStorage.setItem('quizSettings', JSON.stringify(settings));
            
            // Trigger storage event for cross-page updates
            window.dispatchEvent(new Event('settingsUpdated'));
            
            return true;
        } catch (e) {
            console.error('Error saving settings:', e);
            return false;
        }
    },

    // Save selected themes
    saveThemes(themes) {
        try {
            localStorage.setItem('selectedThemes', JSON.stringify(themes));
            window.dispatchEvent(new Event('settingsUpdated'));
            return true;
        } catch (e) {
            console.error('Error saving themes:', e);
            return false;
        }
    },

    // Get current settings
    getSettings() {
        return this.loadSettings();
    },

    // Get time limit
    getTimeLimit() {
        return this.loadSettings().timeLimit;
    },

    // Get question count
    getQuestionCount() {
        return this.loadSettings().questionCount;
    },

    // Get selected themes
    getSelectedThemes() {
        return this.loadSettings().selectedThemes;
    },

    // Get theme count
    getThemeCount() {
        return this.getSelectedThemes().length;
    },

    // Update UI badges on any page
    updateBadges() {
        const settings = this.loadSettings();
        
        // Update time display
        const timeDisplay = document.getElementById('timeDisplay');
        if (timeDisplay) {
            timeDisplay.textContent = settings.timeLimit + 's';
        }

        // Update questions display
        const questionsDisplay = document.getElementById('questionsDisplay');
        if (questionsDisplay) {
            questionsDisplay.textContent = settings.questionCount;
        }

        // Update theme count
        const themeCount = document.getElementById('themeCount');
        if (themeCount) {
            themeCount.textContent = settings.selectedThemes.length || 0;
        }
    },

    // Update all displays (sliders, values, badges)
    updateAllDisplays(timeLimit, questionCount) {
        // Update sliders
        const timeSlider = document.getElementById('timeSlider');
        if (timeSlider) timeSlider.value = timeLimit;

        const questionSlider = document.getElementById('questionSlider');
        if (questionSlider) questionSlider.value = questionCount;

        // Update displayed values
        const timeValue = document.getElementById('timeValue');
        if (timeValue) timeValue.textContent = timeLimit + 's';

        const questionValue = document.getElementById('questionValue');
        if (questionValue) questionValue.textContent = questionCount;

        // Update badges
        this.updateBadges();
    },

    // Reset to defaults
    resetToDefaults() {
        this.saveSettings(this.defaults.timeLimit, this.defaults.questionCount);
        this.updateAllDisplays(this.defaults.timeLimit, this.defaults.questionCount);
    },

    // Initialize settings on page load
    init() {
        const settings = this.loadSettings();
        this.updateAllDisplays(settings.timeLimit, settings.questionCount);
        
        // Listen for settings changes from other tabs/windows
        window.addEventListener('storage', (e) => {
            if (e.key === 'quizSettings' || e.key === 'selectedThemes') {
                this.updateBadges();
            }
        });

        // Listen for local settings updates
        window.addEventListener('settingsUpdated', () => {
            this.updateBadges();
        });
    }
};

// Auto-initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => SettingsManager.init());
} else {
    SettingsManager.init();
}

// Export for use in other scripts
window.SettingsManager = SettingsManager;
