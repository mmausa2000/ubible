/**
 * Bible Books Dictionary & Smart Parser
 * Supports multiple languages for verse parsing
 */

const BibleBooks = {
    // English Old Testament
    'Genesis': true,
    'Exodus': true,
    'Leviticus': true,
    'Numbers': true,
    'Deuteronomy': true,
    'Joshua': true,
    'Judges': true,
    'Ruth': true,
    '1 Samuel': true,
    '2 Samuel': true,
    '1 Kings': true,
    '2 Kings': true,
    '1 Chronicles': true,
    '2 Chronicles': true,
    'Ezra': true,
    'Nehemiah': true,
    'Esther': true,
    'Job': true,
    'Psalms': true,
    'Psalm': true,
    'Proverbs': true,
    'Ecclesiastes': true,
    'Song of Solomon': true,
    'Song of Songs': true,
    'Isaiah': true,
    'Jeremiah': true,
    'Lamentations': true,
    'Ezekiel': true,
    'Daniel': true,
    'Hosea': true,
    'Joel': true,
    'Amos': true,
    'Obadiah': true,
    'Jonah': true,
    'Micah': true,
    'Nahum': true,
    'Habakkuk': true,
    'Zephaniah': true,
    'Haggai': true,
    'Zechariah': true,
    'Malachi': true,

    // English New Testament
    'Matthew': true,
    'Mark': true,
    'Luke': true,
    'John': true,
    'Acts': true,
    'Romans': true,
    '1 Corinthians': true,
    '2 Corinthians': true,
    'Galatians': true,
    'Ephesians': true,
    'Philippians': true,
    'Colossians': true,
    '1 Thessalonians': true,
    '2 Thessalonians': true,
    '1 Timothy': true,
    '2 Timothy': true,
    'Titus': true,
    'Philemon': true,
    'Hebrews': true,
    'James': true,
    '1 Peter': true,
    '2 Peter': true,
    '1 John': true,
    '2 John': true,
    '3 John': true,
    'Jude': true,
    'Revelation': true,

    // Swahili Old Testament
    'Mwanzo': true,
    'Kutoka': true,
    'Mambo ya Walawi': true,
    'Hesabu': true,
    'Kumbukumbu la Torati': true,
    'Yoshua': true,
    'Waamuzi': true,
    'Ruthu': true,
    '1 Samweli': true,
    '2 Samweli': true,
    '1 Wafalme': true,
    '2 Wafalme': true,
    '1 Mambo ya Nyakati': true,
    '2 Mambo ya Nyakati': true,
    'Ezra': true,
    'Nehemia': true,
    'Esta': true,
    'Ayubu': true,
    'Zaburi': true,
    'Mithali': true,
    'Mhubiri': true,
    'Wimbo Ulio Bora': true,
    'Isaya': true,
    'Yeremia': true,
    'Maombolezo': true,
    'Ezekieli': true,
    'Danieli': true,
    'Hosea': true,
    'Yoeli': true,
    'Amosi': true,
    'Obadia': true,
    'Yona': true,
    'Mika': true,
    'Nahumu': true,
    'Habakuki': true,
    'Sefania': true,
    'Hagai': true,
    'Zekaria': true,
    'Malaki': true,

    // Swahili New Testament
    'Mathayo': true,
    'Marko': true,
    'Luka': true,
    'Yohana': true,
    'Matendo': true,
    'Matendo ya Mitume': true,
    'Warumi': true,
    '1 Wakorintho': true,
    '2 Wakorintho': true,
    'Wagalatia': true,
    'Waefeso': true,
    'Wafilipi': true,
    'Wakolosai': true,
    '1 Wathesalonike': true,
    '2 Wathesalonike': true,
    '1 Timotheo': true,
    '2 Timotheo': true,
    'Tito': true,
    'Filemoni': true,
    'Waebrania': true,
    'Yakobo': true,
    '1 Petro': true,
    '2 Petro': true,
    '1 Yohana': true,
    '2 Yohana': true,
    '3 Yohana': true,
    'Yuda': true,
    'Ufunuo': true,

    // French
    'Genèse': true,
    'Exode': true,
    'Lévitique': true,
    'Nombres': true,
    'Deutéronome': true,
    'Josué': true,
    'Juges': true,
    'Ruth': true,
    'Psaumes': true,
    'Matthieu': true,
    'Marc': true,
    'Luc': true,
    'Jean': true,
    'Romains': true,
    'Apocalypse': true,

    // Spanish
    'Génesis': true,
    'Éxodo': true,
    'Levítico': true,
    'Números': true,
    'Deuteronomio': true,
    'Josué': true,
    'Jueces': true,
    'Salmos': true,
    'Mateo': true,
    'Marcos': true,
    'Lucas': true,
    'Juan': true,
    'Romanos': true,
    'Apocalipsis': true
};

/**
 * Find book name at the start of text
 */
function findBookInText(text) {
    // Sort books by length (longest first) to match multi-word books first
    const sortedBooks = Object.keys(BibleBooks).sort((a, b) => b.length - a.length);
    
    for (const book of sortedBooks) {
        if (text.startsWith(book + ' ') || text.startsWith(book + '\t')) {
            return book;
        }
    }
    return null;
}

/**
 * Calculate Levenshtein distance for fuzzy matching
 */
function levenshteinDistance(str1, str2) {
    const matrix = [];
    
    for (let i = 0; i <= str2.length; i++) {
        matrix[i] = [i];
    }
    
    for (let j = 0; j <= str1.length; j++) {
        matrix[0][j] = j;
    }
    
    for (let i = 1; i <= str2.length; i++) {
        for (let j = 1; j <= str1.length; j++) {
            if (str2.charAt(i - 1) === str1.charAt(j - 1)) {
                matrix[i][j] = matrix[i - 1][j - 1];
            } else {
                matrix[i][j] = Math.min(
                    matrix[i - 1][j - 1] + 1,
                    matrix[i][j - 1] + 1,
                    matrix[i - 1][j] + 1
                );
            }
        }
    }
    
    return matrix[str2.length][str1.length];
}

/**
 * Fuzzy match book name (handles typos and variations)
 */
function fuzzyMatchBook(text, threshold = 2) {
    const words = text.split(/\s+/);
    const possibleBook = words.slice(0, 4).join(' '); // Check first 4 words
    
    let bestMatch = null;
    let bestDistance = Infinity;
    
    for (const book of Object.keys(BibleBooks)) {
        const bookLower = book.toLowerCase();
        const textLower = possibleBook.toLowerCase();
        
        // Exact match first
        if (textLower.startsWith(bookLower)) {
            return book;
        }
        
        // Fuzzy match
        const distance = levenshteinDistance(
            textLower.substring(0, bookLower.length),
            bookLower
        );
        
        if (distance < bestDistance && distance <= threshold) {
            bestDistance = distance;
            bestMatch = book;
        }
    }
    
    return bestMatch;
}

/**
 * Find book with fuzzy matching support
 */
function findBookInTextSmart(text) {
    // Try exact match first
    const exact = findBookInText(text);
    if (exact) return exact;
    
    // Try fuzzy match
    return fuzzyMatchBook(text);
}

/**
 * Smart parse - recognizes book names and extracts reference + text
 */
function smartParse(line) {
    const book = findBookInTextSmart(line);
    if (!book) return null;
    
    // Escape special regex characters in book name
    const escapedBook = book.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    
    // Match: BookName Chapter:Verse(-Verse) separator Text
    const refMatch = line.match(
        new RegExp(`^(${escapedBook}\\s+\\d+:\\d+(?:-\\d+)?)\\s*[—\\-–:]*\\s*(.+)$`, 'i')
    );
    
    if (refMatch) {
        return {
            ref: refMatch[1].trim(),
            text: refMatch[2].trim()
        };
    }
    
    return null;
}

/**
 * Check if a string is a valid book name
 */
function isValidBook(bookName) {
    return BibleBooks[bookName] === true;
}

/**
 * Get all book names
 */
function getAllBooks() {
    return Object.keys(BibleBooks);
}

/**
 * Get books by language
 */
function getBooksByLanguage(lang) {
    const languages = {
        english: ['Genesis', 'Exodus', 'Leviticus', 'Matthew', 'Mark', 'Luke', 'John'],
        swahili: ['Mwanzo', 'Kutoka', 'Mambo ya Walawi', 'Mathayo', 'Marko', 'Luka', 'Yohana'],
        french: ['Genèse', 'Exode', 'Matthieu', 'Marc', 'Luc', 'Jean'],
        spanish: ['Génesis', 'Éxodo', 'Mateo', 'Marcos', 'Lucas', 'Juan']
    };
    
    return languages[lang.toLowerCase()] || [];
}

// Export for use in other files
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        BibleBooks,
        findBookInText,
        findBookInTextSmart,
        fuzzyMatchBook,
        smartParse,
        isValidBook,
        getAllBooks,
        getBooksByLanguage
    };
}
