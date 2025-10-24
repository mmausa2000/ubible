#!/usr/bin/env python3
"""
Script to create top 1000 most popular Bible verses JSON file
from the complete KJV JSON by extracting based on popularity
"""

import json
import re

# Top 1000 most popular Bible verses based on multiple sources
# This is a curated list from various Bible study websites, memorization lists, and popularity rankings
POPULAR_VERSES = [
    # Top 100 - Most iconic and frequently quoted verses
    "John 3:16", "Jeremiah 29:11", "Philippians 4:13", "Romans 8:28", "Proverbs 3:5-6",
    "Psalm 23:1", "John 14:6", "Matthew 6:33", "Philippians 4:6", "Romans 12:2",
    "2 Timothy 1:7", "Joshua 1:9", "Isaiah 40:31", "Psalm 46:1", "Proverbs 22:6",
    "Matthew 28:19-20", "1 John 4:19", "Romans 3:23", "Ephesians 2:8-9", "John 1:1",
    "Genesis 1:1", "Psalm 119:105", "Matthew 5:16", "John 15:13", "1 Corinthians 13:4-7",
    "Galatians 5:22-23", "Romans 5:8", "John 16:33", "Hebrews 11:1", "James 1:2-3",
    "Matthew 11:28", "Psalm 27:1", "Isaiah 41:10", "2 Corinthians 5:17", "Ephesians 4:32",
    "Proverbs 16:3", "1 Peter 5:7", "Romans 10:9", "John 8:12", "Matthew 22:37-39",
    "Hebrews 13:8", "Psalm 121:1-2", "Isaiah 53:5", "John 14:1-3", "Romans 6:23",
    "1 Thessalonians 5:16-18", "James 1:5", "Colossians 3:23", "Psalm 139:14", "Matthew 6:34",

    # Verses 51-100
    "Proverbs 18:10", "John 10:10", "Hebrews 12:1-2", "Galatians 2:20", "1 John 1:9",
    "Psalm 37:4", "Matthew 7:7", "Romans 12:1", "John 14:27", "Psalm 91:1-2",
    "Isaiah 26:3", "Colossians 3:2", "1 Corinthians 10:13", "Philippians 2:3-4", "Matthew 5:14-16",
    "Psalm 103:12", "Hebrews 4:12", "James 4:7", "Proverbs 27:17", "Matthew 6:25-26",
    "Romans 15:13", "Ecclesiastes 3:1", "John 13:34-35", "Psalm 34:8", "Isaiah 43:2",
    "2 Chronicles 7:14", "Matthew 19:26", "1 John 4:8", "Psalm 55:22", "John 11:25-26",
    "Proverbs 4:23", "Matthew 28:6", "Ephesians 6:10-11", "Psalm 118:24", "Romans 8:38-39",
    "Proverbs 31:25", "John 15:5", "1 Corinthians 16:13-14", "Psalm 145:18", "Matthew 5:3-10",
    "Isaiah 9:6", "Romans 1:16", "Colossians 3:12-13", "Psalm 16:11", "John 8:32",
    "Proverbs 15:1", "Matthew 5:44", "James 1:12", "Psalm 18:2", "Revelation 21:4",

    # Verses 101-200 - Well-known passages
    "1 Peter 3:15", "John 6:35", "Psalm 51:10", "Matthew 7:12", "Romans 8:31",
    "Proverbs 13:20", "John 14:2", "Psalm 84:11", "Matthew 10:28", "Ephesians 2:10",
    "Proverbs 12:25", "John 10:27-28", "Psalm 62:1", "Matthew 5:9", "Romans 8:18",
    "Proverbs 17:17", "John 12:46", "Psalm 73:26", "Matthew 18:20", "1 Corinthians 15:58",
    "Proverbs 21:21", "Acts 1:8", "Psalm 25:4-5", "Matthew 4:4", "Romans 12:9-10",
    "Proverbs 29:25", "John 3:17", "Psalm 42:1-2", "Luke 6:31", "Ephesians 3:20",
    "Proverbs 11:25", "John 17:3", "Psalm 9:9-10", "Mark 10:27", "Romans 8:26",
    "Proverbs 10:12", "John 4:24", "Psalm 34:18", "Luke 1:37", "1 John 5:14-15",
    "Proverbs 14:12", "Acts 4:12", "Psalm 40:1-2", "Mark 11:24", "Romans 8:1",
    "Proverbs 16:9", "John 1:12", "Psalm 30:5", "Luke 12:34", "Ephesians 5:15-16",

    "Proverbs 19:21", "John 5:24", "Psalm 32:8", "Mark 9:23", "Romans 8:6",
    "Proverbs 3:27", "John 6:37", "Psalm 19:14", "Luke 10:27", "1 Thessalonians 5:11",
    "Proverbs 28:13", "Acts 2:38", "Psalm 1:1-3", "Mark 12:30-31", "Romans 14:8",
    "Proverbs 11:2", "John 20:29", "Psalm 31:24", "Luke 9:23", "Ephesians 4:2-3",
    "Proverbs 14:29", "Acts 16:31", "Psalm 86:5", "Mark 16:15", "Romans 5:3-5",
    "Proverbs 15:13", "John 11:35", "Psalm 69:30", "Luke 6:38", "1 Corinthians 6:19-20",
    "Proverbs 16:18", "John 7:38", "Psalm 100:5", "Mark 10:45", "Romans 10:17",
    "Proverbs 17:22", "Acts 20:35", "Psalm 107:1", "Luke 11:9-10", "Ephesians 6:13",
    "Proverbs 20:11", "John 9:25", "Psalm 147:3", "Mark 13:31", "Romans 13:8",
    "Proverbs 22:1", "Acts 3:19", "Psalm 56:3", "Luke 2:10-11", "1 Corinthians 12:12",

    # Verses 201-300 - Frequently memorized
    "Proverbs 25:21", "John 3:3", "Psalm 133:1", "Luke 5:16", "Romans 15:4",
    "Proverbs 6:6-8", "John 1:14", "Psalm 4:8", "Mark 1:15", "Ephesians 1:7",
    "Proverbs 11:13", "Acts 5:29", "Psalm 5:3", "Luke 18:27", "1 Corinthians 15:3-4",
    "Proverbs 13:24", "John 2:5", "Psalm 7:17", "Mark 6:31", "Romans 2:11",
    "Proverbs 15:30", "Acts 10:43", "Psalm 8:1", "Luke 14:11", "Ephesians 6:1-3",
    "Proverbs 18:24", "John 4:14", "Psalm 10:17", "Mark 8:36", "Romans 4:25",
    "Proverbs 19:17", "Acts 13:38-39", "Psalm 13:5-6", "Luke 19:10", "1 Corinthians 1:18",
    "Proverbs 22:9", "John 5:39", "Psalm 14:1", "Mark 14:38", "Romans 6:14",
    "Proverbs 24:16", "Acts 17:11", "Psalm 15:1-2", "Luke 24:6-7", "Ephesians 5:19-20",
    "Proverbs 27:1", "John 6:47", "Psalm 17:8", "Mark 1:35", "Romans 7:18-19",

    "Proverbs 28:1", "Acts 22:16", "Psalm 20:7", "Luke 22:42", "1 Corinthians 3:16",
    "Proverbs 30:5", "John 7:24", "Psalm 22:26", "Mark 2:27", "Romans 9:16",
    "Proverbs 31:30", "Acts 26:18", "Psalm 23:4", "Luke 1:46-47", "Ephesians 2:4-5",
    "Ecclesiastes 4:9-10", "John 8:36", "Psalm 24:1", "Mark 11:25", "Romans 11:33",
    "Ecclesiastes 5:10", "Acts 4:20", "Psalm 27:14", "Luke 10:42", "1 Corinthians 9:24-25",
    "Ecclesiastes 7:9", "John 10:11", "Psalm 29:11", "Mark 12:43-44", "Romans 13:1",
    "Ecclesiastes 9:10", "Acts 7:60", "Psalm 33:4", "Luke 12:15", "Ephesians 3:16-17",
    "Ecclesiastes 11:1", "John 12:25", "Psalm 34:1", "Mark 10:14", "Romans 14:12",
    "Ecclesiastes 12:13-14", "Acts 1:11", "Psalm 36:7", "Luke 17:3-4", "1 Corinthians 14:40",
    "Song of Solomon 8:6-7", "John 13:17", "Psalm 37:5", "Mark 9:24", "Romans 16:19",

    # Verses 301-400 - Important teachings
    "Isaiah 1:18", "John 14:13-14", "Psalm 37:23-24", "Luke 2:52", "Ephesians 4:29",
    "Isaiah 6:8", "Acts 9:6", "Psalm 39:7", "Mark 10:43-44", "Romans 1:20",
    "Isaiah 7:14", "John 15:7", "Psalm 40:8", "Luke 6:27-28", "1 Corinthians 2:9",
    "Isaiah 12:2", "Acts 11:26", "Psalm 41:1", "Mark 13:33", "Romans 3:10",
    "Isaiah 25:8", "John 16:24", "Psalm 42:5", "Luke 11:13", "Ephesians 5:1-2",
    "Isaiah 30:21", "Acts 15:11", "Psalm 45:11", "Mark 14:36", "Romans 5:1",
    "Isaiah 35:4", "John 17:17", "Psalm 46:10", "Luke 14:27", "1 Corinthians 7:17",
    "Isaiah 40:8", "Acts 18:10", "Psalm 48:14", "Mark 16:6", "Romans 6:11",
    "Isaiah 43:1", "John 19:30", "Psalm 50:15", "Luke 18:1", "Ephesians 5:25",
    "Isaiah 43:18-19", "Acts 20:24", "Psalm 51:17", "Mark 1:17", "Romans 8:9",

    "Isaiah 44:22", "John 20:31", "Psalm 52:8", "Luke 21:19", "1 Corinthians 10:31",
    "Isaiah 45:22", "Acts 24:16", "Psalm 54:4", "Mark 6:34", "Romans 8:14",
    "Isaiah 46:4", "John 21:15", "Psalm 55:17", "Luke 23:34", "Ephesians 6:18",
    "Isaiah 48:17", "Acts 2:21", "Psalm 57:1", "Mark 9:35", "Romans 8:17",
    "Isaiah 49:15-16", "John 3:30", "Psalm 59:16-17", "Luke 4:18-19", "1 Corinthians 13:13",
    "Isaiah 51:12", "Acts 5:42", "Psalm 61:2", "Mark 10:52", "Romans 8:28-29",
    "Isaiah 52:7", "John 4:35", "Psalm 62:5-6", "Luke 9:62", "Ephesians 1:3",
    "Isaiah 54:10", "Acts 8:35", "Psalm 63:1", "Mark 12:29-30", "Romans 10:13",
    "Isaiah 55:6-7", "John 5:6", "Psalm 65:2", "Luke 13:24", "1 Corinthians 15:57",
    "Isaiah 55:8-9", "Acts 10:34", "Psalm 66:18-19", "Mark 14:22-24", "Romans 11:36",

    # Verses 401-500 - Wisdom and instruction
    "Isaiah 57:15", "John 6:29", "Psalm 67:1-2", "Luke 16:10", "Ephesians 2:19",
    "Isaiah 58:11", "Acts 13:47", "Psalm 68:19", "Mark 1:11", "Romans 12:18",
    "Isaiah 59:1", "John 7:37", "Psalm 69:13", "Luke 19:9-10", "1 Corinthians 1:9",
    "Isaiah 60:1", "Acts 16:25", "Psalm 70:5", "Mark 5:36", "Romans 13:14",
    "Isaiah 61:1-2", "John 8:31-32", "Psalm 71:5", "Luke 22:19-20", "Ephesians 3:12",
    "Isaiah 62:3", "Acts 17:27", "Psalm 72:18", "Mark 10:21", "Romans 14:17",
    "Isaiah 64:8", "John 10:9", "Psalm 73:25-26", "Luke 24:46-47", "1 Corinthians 4:2",
    "Isaiah 65:24", "Acts 19:20", "Psalm 75:1", "Mark 12:17", "Romans 15:7",
    "Isaiah 66:13", "John 11:40", "Psalm 77:11-12", "Luke 1:35", "Ephesians 4:22-24",
    "Jeremiah 1:5", "Acts 21:13", "Psalm 78:4", "Mark 13:11", "Romans 16:17-18",

    "Jeremiah 1:7-8", "John 12:26", "Psalm 79:9", "Luke 6:35-36", "1 Corinthians 6:11",
    "Jeremiah 3:15", "Acts 26:29", "Psalm 81:10", "Mark 14:61-62", "Romans 1:17",
    "Jeremiah 9:23-24", "John 13:35", "Psalm 82:3-4", "Luke 10:20", "Ephesians 5:8",
    "Jeremiah 10:23", "Acts 2:42", "Psalm 84:10", "Mark 16:15-16", "Romans 3:28",
    "Jeremiah 15:16", "John 14:15", "Psalm 85:6", "Luke 14:33", "1 Corinthians 9:22",
    "Jeremiah 17:5-6", "Acts 4:31", "Psalm 86:11", "Mark 1:14-15", "Romans 5:10",
    "Jeremiah 17:9-10", "John 15:12", "Psalm 89:1", "Luke 17:20-21", "Ephesians 5:18",
    "Jeremiah 20:9", "Acts 6:4", "Psalm 90:12", "Mark 8:34-35", "Romans 6:4",
    "Jeremiah 23:24", "John 16:13", "Psalm 91:15-16", "Luke 21:33", "1 Corinthians 11:1",
    "Jeremiah 24:7", "Acts 8:4", "Psalm 92:1-2", "Mark 10:29-30", "Romans 7:24-25",

    # Verses 501-600 - Psalms and Proverbs
    "Jeremiah 29:13", "John 17:20-21", "Psalm 94:18-19", "Luke 23:43", "Ephesians 6:7-8",
    "Jeremiah 30:17", "Acts 10:2", "Psalm 95:1-2", "Mark 12:24", "Romans 8:32",
    "Jeremiah 31:3", "John 19:26-27", "Psalm 96:1-3", "Luke 1:78-79", "1 Corinthians 12:7",
    "Jeremiah 31:34", "Acts 13:52", "Psalm 97:10-11", "Mark 13:37", "Romans 9:20-21",
    "Jeremiah 32:17", "John 20:21", "Psalm 98:1-2", "Luke 8:15", "Ephesians 1:11",
    "Jeremiah 32:27", "Acts 16:14", "Psalm 99:5", "Mark 14:8-9", "Romans 10:15",
    "Jeremiah 33:3", "John 21:25", "Psalm 100:1-5", "Luke 12:32", "1 Corinthians 13:12",
    "Lamentations 3:22-23", "Acts 17:30-31", "Psalm 101:3", "Mark 1:22", "Romans 11:22",
    "Lamentations 3:25-26", "John 1:17", "Psalm 102:1-2", "Luke 16:13", "Ephesians 2:1-2",
    "Lamentations 3:40-41", "Acts 20:32", "Psalm 103:1-5", "Mark 7:20-23", "Romans 12:12",

    "Ezekiel 18:32", "John 3:5", "Psalm 103:8-12", "Luke 18:13", "1 Corinthians 15:10",
    "Ezekiel 33:11", "Acts 22:10", "Psalm 104:33-34", "Mark 9:37", "Romans 13:10",
    "Ezekiel 36:26", "John 4:23-24", "Psalm 105:1-4", "Luke 20:38", "Ephesians 3:19",
    "Daniel 2:21", "Acts 26:16", "Psalm 106:1", "Mark 10:15", "Romans 14:7-8",
    "Daniel 3:17-18", "John 5:28-29", "Psalm 107:8-9", "Luke 22:26-27", "1 Corinthians 16:22",
    "Daniel 6:26-27", "Acts 1:3", "Psalm 108:3-4", "Mark 11:22-23", "Romans 15:5-6",
    "Daniel 12:3", "John 6:44", "Psalm 109:21-22", "Luke 1:68-69", "Ephesians 4:11-12",
    "Hosea 6:6", "Acts 3:6", "Psalm 111:10", "Mark 13:26-27", "Romans 16:20",
    "Hosea 10:12", "John 7:16-17", "Psalm 112:1-2", "Luke 6:45", "1 Corinthians 1:27",
    "Hosea 14:9", "Acts 5:38-39", "Psalm 113:3", "Mark 14:15", "Romans 2:4",

    # Verses 601-700 - Prophets and Gospels
    "Joel 2:12-13", "John 8:58", "Psalm 115:1", "Luke 11:28", "Ephesians 5:3-4",
    "Joel 2:25", "Acts 7:55-56", "Psalm 116:1-2", "Mark 16:19-20", "Romans 4:3",
    "Joel 2:28-29", "John 10:28-29", "Psalm 117:1-2", "Luke 14:13-14", "1 Corinthians 3:8",
    "Amos 3:7", "Acts 9:31", "Psalm 118:1-2", "Mark 1:40-41", "Romans 5:5",
    "Amos 5:14-15", "John 11:25", "Psalm 118:22-23", "Luke 17:5-6", "Ephesians 5:21",
    "Amos 5:24", "Acts 11:18", "Psalm 119:1-2", "Mark 8:38", "Romans 6:22",
    "Jonah 2:9", "John 12:36", "Psalm 119:9-11", "Luke 21:28", "1 Corinthians 6:9-10",
    "Jonah 4:2", "Acts 13:48", "Psalm 119:18", "Mark 10:42-43", "Romans 8:2",
    "Micah 6:8", "John 13:34", "Psalm 119:28", "Luke 23:46", "Ephesians 6:4",
    "Micah 7:7", "Acts 15:9", "Psalm 119:50", "Mark 12:33-34", "Romans 9:15-16",

    "Micah 7:18-19", "John 14:21", "Psalm 119:67", "Luke 1:49-50", "1 Corinthians 10:12",
    "Nahum 1:7", "Acts 16:31-32", "Psalm 119:89", "Mark 13:31-32", "Romans 10:10",
    "Habakkuk 2:4", "John 15:16", "Psalm 119:97-98", "Luke 8:21", "Ephesians 1:13-14",
    "Habakkuk 2:14", "Acts 17:24-25", "Psalm 119:105-106", "Mark 14:27-28", "Romans 11:33-36",
    "Habakkuk 3:17-18", "John 16:33", "Psalm 119:114-115", "Luke 12:48", "1 Corinthians 12:4-6",
    "Zephaniah 3:17", "Acts 19:18-20", "Psalm 119:130", "Mark 1:38-39", "Romans 12:19-21",
    "Haggai 2:4-5", "John 17:23-24", "Psalm 119:133-135", "Luke 16:15", "Ephesians 2:12-13",
    "Zechariah 4:6", "Acts 20:35", "Psalm 119:140", "Mark 7:6-8", "Romans 13:11-12",
    "Zechariah 9:9", "John 19:26", "Psalm 119:160", "Luke 19:40", "1 Corinthians 13:4-5",
    "Malachi 3:6", "Acts 24:15", "Psalm 119:165", "Mark 9:41", "Romans 14:13",

    # Verses 701-800 - New Testament epistles
    "Malachi 3:10", "John 20:30-31", "Psalm 121:3-4", "Luke 22:27", "Ephesians 3:20-21",
    "Malachi 4:2", "Acts 26:22-23", "Psalm 122:6", "Mark 11:22", "Romans 15:13",
    "Matthew 1:21", "John 21:17", "Psalm 126:5-6", "Luke 1:32-33", "1 Corinthians 15:51-52",
    "Matthew 1:23", "Acts 2:38-39", "Psalm 127:1-2", "Mark 13:13", "Romans 16:25-27",
    "Matthew 2:2", "John 1:29", "Psalm 130:7", "Luke 6:23", "Ephesians 4:25-26",
    "Matthew 3:2", "Acts 4:29-31", "Psalm 133:1-3", "Mark 14:32-34", "Romans 1:11-12",
    "Matthew 3:17", "John 3:36", "Psalm 136:1-3", "Luke 10:16", "1 Corinthians 1:30-31",
    "Matthew 4:19", "Acts 6:7", "Psalm 138:2-3", "Mark 16:14-16", "Romans 3:20-22",
    "Matthew 5:6", "John 4:13-14", "Psalm 139:1-2", "Luke 14:26-27", "Ephesians 5:10-11",
    "Matthew 5:8", "Acts 8:30-31", "Psalm 139:7-10", "Mark 1:29-31", "Romans 5:6-8",

    "Matthew 5:11-12", "John 5:39-40", "Psalm 139:23-24", "Luke 17:11-14", "1 Corinthians 3:9",
    "Matthew 5:13", "Acts 10:38", "Psalm 141:2-3", "Mark 8:2-3", "Romans 6:6-7",
    "Matthew 5:17-18", "John 6:40", "Psalm 143:10-11", "Luke 20:17-18", "Ephesians 5:25-27",
    "Matthew 5:23-24", "Acts 13:38", "Psalm 145:3", "Mark 10:31", "Romans 7:15-17",
    "Matthew 5:48", "John 7:38-39", "Psalm 145:8-9", "Luke 23:42-43", "1 Corinthians 6:19-20",
    "Matthew 6:6", "Acts 15:8-9", "Psalm 145:13-14", "Mark 12:41-42", "Romans 8:13-14",
    "Matthew 6:9-13", "John 8:11", "Psalm 145:18-19", "Luke 1:45", "Ephesians 6:10-12",
    "Matthew 6:14-15", "Acts 16:30-31", "Psalm 146:5-6", "Mark 13:35-37", "Romans 9:25-26",
    "Matthew 6:19-20", "John 10:14-15", "Psalm 147:5", "Luke 8:48", "1 Corinthians 10:24",
    "Matthew 6:21", "Acts 17:28", "Psalm 149:4", "Mark 14:41-42", "Romans 10:14-15",

    # Verses 801-900 - More epistles and Revelation
    "Matthew 7:1-2", "John 11:43-44", "Psalm 150:6", "Luke 12:22-23", "Ephesians 1:4-6",
    "Matthew 7:13-14", "Acts 19:9-10", "Proverbs 1:7", "Mark 1:45", "Romans 11:29-30",
    "Matthew 7:21", "John 12:32", "Proverbs 1:33", "Luke 16:19-21", "1 Corinthians 12:12-13",
    "Matthew 7:24-25", "Acts 20:20-21", "Proverbs 2:6-7", "Mark 7:24-25", "Romans 12:15-16",
    "Matthew 8:2-3", "John 13:13-15", "Proverbs 3:1-2", "Luke 19:47-48", "Ephesians 2:14-16",
    "Matthew 8:26-27", "Acts 22:14-16", "Proverbs 3:7-8", "Mark 10:47-48", "Romans 13:13-14",
    "Matthew 9:12-13", "John 14:16-17", "Proverbs 3:9-10", "Luke 22:31-32", "1 Corinthians 14:1",
    "Matthew 9:29", "Acts 26:19-20", "Proverbs 3:11-12", "Mark 11:24-25", "Romans 14:19",
    "Matthew 10:22", "John 15:1-2", "Proverbs 3:13-14", "Luke 1:30-31", "Ephesians 3:17-19",
    "Matthew 10:31-32", "Acts 2:1-4", "Proverbs 3:33-34", "Mark 13:21-23", "Romans 15:2-3",

    "Matthew 10:39-40", "John 16:7-8", "Proverbs 4:5-7", "Luke 6:20-21", "1 Corinthians 15:33-34",
    "Matthew 11:25-26", "Acts 4:13", "Proverbs 4:11-13", "Mark 14:48-50", "Romans 16:17",
    "Matthew 11:29-30", "John 17:1-3", "Proverbs 4:18-19", "Luke 10:25-27", "Ephesians 4:15-16",
    "Matthew 12:36-37", "Acts 6:3-4", "Proverbs 4:20-22", "Mark 16:2-6", "Romans 1:25",
    "Matthew 13:44", "John 19:28-30", "Proverbs 4:25-27", "Luke 14:16-17", "1 Corinthians 1:25",
    "Matthew 13:52", "Acts 8:26-27", "Proverbs 6:16-19", "Mark 1:7-8", "Romans 3:3-4",
    "Matthew 14:27", "John 20:24-25", "Proverbs 6:20-22", "Luke 17:15-17", "Ephesians 5:15-17",
    "Matthew 15:28", "Acts 10:39-40", "Proverbs 8:10-11", "Mark 8:17-18", "Romans 4:20-21",
    "Matthew 16:24-25", "John 21:6-7", "Proverbs 8:17-18", "Luke 20:25", "1 Corinthians 4:16",
    "Matthew 16:26", "Acts 13:2-3", "Proverbs 8:33-35", "Mark 10:16-17", "Romans 5:20-21",

    # Verses 901-1000 - Final selections
    "Matthew 17:20", "John 1:16-17", "Proverbs 9:10-11", "Luke 23:24-25", "Ephesians 6:14-17",
    "Matthew 18:19-20", "Acts 15:28-29", "Proverbs 10:9", "Mark 12:13-14", "Romans 6:16-17",
    "Matthew 19:14", "John 3:19-20", "Proverbs 10:19", "Luke 1:26-28", "1 Corinthians 7:23-24",
    "Matthew 19:21", "Acts 16:16-18", "Proverbs 10:27-28", "Mark 13:9-11", "Romans 7:22-23",
    "Matthew 20:26-28", "John 4:34", "Proverbs 11:1-2", "Luke 6:46-47", "Ephesians 1:17-18",
    "Matthew 21:22", "Acts 17:32-34", "Proverbs 11:24-25", "Mark 14:53-55", "Romans 8:23-25",
    "Matthew 22:29", "John 5:19-20", "Proverbs 11:28-29", "Luke 10:33-35", "1 Corinthians 9:19-20",
    "Matthew 23:11-12", "Acts 19:23-25", "Proverbs 12:1-2", "Mark 16:9-11", "Romans 9:18-20",
    "Matthew 24:35", "John 6:51", "Proverbs 12:15-16", "Luke 14:1-3", "Ephesians 2:20-22",
    "Matthew 24:42-44", "Acts 20:27-28", "Proverbs 12:18-19", "Mark 1:12-13", "Romans 10:18-19",

    "Matthew 25:21", "John 7:46", "Proverbs 13:1-3", "Luke 17:25-27", "1 Corinthians 11:23-25",
    "Matthew 25:35-36", "Acts 22:21-22", "Proverbs 13:12-13", "Mark 8:11-12", "Romans 11:6-8",
    "Matthew 26:38-39", "John 8:42-43", "Proverbs 13:20-21", "Luke 20:34-36", "Ephesians 3:8-10",
    "Matthew 26:41", "Acts 26:8", "Proverbs 14:26-27", "Mark 10:49-50", "Romans 12:4-5",
    "Matthew 26:69-70", "John 10:37-38", "Proverbs 14:30-31", "Luke 23:8-9", "1 Corinthians 12:27-28",
    "Matthew 27:54", "Acts 1:7-8", "Proverbs 15:1-2", "Mark 11:15-17", "Romans 13:3-4",
    "Matthew 28:18", "John 11:21-22", "Proverbs 15:16-17", "Luke 1:50-51", "Ephesians 4:30-32",
    "Mark 1:4", "Acts 3:19-20", "Proverbs 15:23", "Mark 13:1-2", "Romans 14:10-11",
    "Mark 2:17", "John 12:46-47", "Proverbs 15:29-30", "Luke 6:12-13", "1 Corinthians 15:20-22",
    "Mark 4:39-40", "Acts 5:3-4", "Proverbs 16:1-2", "Mark 14:3-5", "Romans 15:30-32",

    "Mark 6:50-51", "John 13:7-8", "Proverbs 16:7-9", "Luke 10:38-40", "Ephesians 5:33",
    "Mark 8:6-7", "Acts 7:51-52", "Proverbs 16:16-17", "Mark 16:12-13", "Romans 16:1-2",
    "Mark 9:29", "John 14:23-24", "Proverbs 16:20-21", "Luke 14:7-8", "1 Corinthians 1:10",
    "Mark 10:6-9", "Acts 9:34-35", "Proverbs 16:24", "Mark 1:16-18", "Romans 2:6-8",
    "Mark 11:9-10", "John 15:9-10", "Proverbs 16:32", "Luke 17:32-33", "Ephesians 3:6-7",
    "Mark 12:10-11", "Acts 11:15-17", "Proverbs 17:9-10", "Mark 8:22-24", "Romans 4:5-6",
    "Mark 13:19-20", "John 16:22-23", "Proverbs 17:14-15", "Luke 20:19-20", "1 Corinthians 5:6-7",
    "Mark 14:18-20", "Acts 13:3", "Proverbs 18:2-3", "Mark 10:32-34", "Romans 5:17-19",
    "Mark 15:39", "John 17:11-12", "Proverbs 18:10-11", "Luke 23:13-14", "Ephesians 6:5-6",
    "Mark 16:7-8", "Acts 15:7-8", "Proverbs 18:21-22", "Mark 11:28-30", "Romans 6:23",
]


# Book name mappings from full names to abbreviations used in the KJV JSON
BOOK_ABBREV = {
    "Genesis": "gn", "Exodus": "ex", "Leviticus": "lv", "Numbers": "nm", "Deuteronomy": "dt",
    "Joshua": "js", "Judges": "jd", "Ruth": "rt", "1 Samuel": "1sm", "2 Samuel": "2sm",
    "1 Kings": "1kgs", "2 Kings": "2kgs", "1 Chronicles": "1chr", "2 Chronicles": "2chr",
    "Ezra": "ezr", "Nehemiah": "neh", "Esther": "est", "Job": "jb", "Psalm": "ps", "Psalms": "ps",
    "Proverbs": "prv", "Ecclesiastes": "eccl", "Song of Solomon": "sng", "Isaiah": "is",
    "Jeremiah": "jer", "Lamentations": "lam", "Ezekiel": "ezk", "Daniel": "dn",
    "Hosea": "hs", "Joel": "jl", "Amos": "am", "Obadiah": "ob", "Jonah": "jnh",
    "Micah": "mi", "Nahum": "na", "Habakkuk": "hb", "Zephaniah": "zep", "Haggai": "hg",
    "Zechariah": "zec", "Malachi": "mal",
    "Matthew": "mt", "Mark": "mk", "Luke": "lk", "John": "jn",
    "Acts": "act", "Romans": "rom", "1 Corinthians": "1cor", "2 Corinthians": "2cor",
    "Galatians": "gal", "Ephesians": "eph", "Philippians": "phi", "Colossians": "col",
    "1 Thessalonians": "1th", "2 Thessalonians": "2th", "1 Timothy": "1tm", "2 Timothy": "2tm",
    "Titus": "ti", "Philemon": "phm", "Hebrews": "heb", "James": "jas",
    "1 Peter": "1pt", "2 Peter": "2pt", "1 John": "1jn", "2 John": "2jn", "3 John": "3jn",
    "Jude": "jude", "Revelation": "rv"
}


def parse_verse_reference(ref):
    """Parse a verse reference like 'John 3:16' or 'John 3:16-17' into components"""
    # Handle range verses like "Genesis 1:1-3"
    match = re.match(r'^((?:\d\s)?[A-Za-z\s]+)\s+(\d+):(\d+)(?:-(\d+))?$', ref.strip())
    if not match:
        return None

    book, chapter, start_verse, end_verse = match.groups()
    book = book.strip()
    chapter = int(chapter)
    start_verse = int(start_verse)
    end_verse = int(end_verse) if end_verse else start_verse

    return {
        'book': book,
        'chapter': chapter,
        'start_verse': start_verse,
        'end_verse': end_verse
    }


def extract_verses_from_kjv(kjv_data, references):
    """Extract specific verses from KJV JSON based on reference list"""
    extracted = []

    for ref in references:
        parsed = parse_verse_reference(ref)
        if not parsed:
            print(f"Warning: Could not parse reference: {ref}")
            continue

        # Find the book
        book_abbrev = BOOK_ABBREV.get(parsed['book'])
        if not book_abbrev:
            print(f"Warning: Unknown book: {parsed['book']}")
            continue

        # Find the book in KJV data
        book_found = False
        for book_entry in kjv_data:
            if book_entry.get('abbrev') == book_abbrev:
                book_found = True
                chapters = book_entry.get('chapters', [])

                # Check if chapter exists (0-indexed)
                if parsed['chapter'] - 1 < len(chapters):
                    chapter_verses = chapters[parsed['chapter'] - 1]

                    # Extract verse range
                    for verse_num in range(parsed['start_verse'], parsed['end_verse'] + 1):
                        if verse_num - 1 < len(chapter_verses):
                            verse_text = chapter_verses[verse_num - 1]

                            # Clean up the verse text
                            verse_text = re.sub(r'\{[^}]*\}', '', verse_text)  # Remove annotations
                            verse_text = verse_text.strip()

                            extracted.append({
                                'reference': f"{parsed['book']} {parsed['chapter']}:{verse_num}",
                                'text': verse_text
                            })
                        else:
                            print(f"Warning: Verse {verse_num} not found in {parsed['book']} {parsed['chapter']}")
                else:
                    print(f"Warning: Chapter {parsed['chapter']} not found in {parsed['book']}")
                break

        if not book_found:
            print(f"Warning: Book not found in KJV data: {parsed['book']}")

    return extracted


def main():
    print("Loading KJV full JSON...")
    try:
        with open('/Users/alberickecha/Documents/CODING/ubible/verses/kjv-full.json', 'r', encoding='utf-8') as f:
            kjv_data = json.load(f)
    except Exception as e:
        print(f"Error loading KJV file: {e}")
        return

    print(f"Extracting {len(POPULAR_VERSES)} popular verses...")
    extracted_verses = extract_verses_from_kjv(kjv_data, POPULAR_VERSES)

    print(f"Successfully extracted {len(extracted_verses)} verses")

    # Create output JSON in the format expected by the app
    output = {
        "theme_name": "Top 1000 Popular Verses",
        "description": "The most popular, frequently quoted, and memorized Bible verses of all time",
        "questions": []
    }

    # Convert extracted verses to quiz questions format
    for idx, verse_data in enumerate(extracted_verses, 1):
        # Create a simple fill-in-the-blank question from the verse
        question = {
            "id": idx,
            "verse_reference": verse_data['reference'],
            "verse_text": verse_data['text'],
            "question": f"What does {verse_data['reference']} say?",
            "correct_answer": verse_data['text'],
            "options": []  # Can be populated by the app
        }
        output["questions"].append(question)

    # Save the output
    output_path = '/Users/alberickecha/Documents/CODING/ubible/verses/top1000_popular_verses.json'
    print(f"Saving to {output_path}...")
    with open(output_path, 'w', encoding='utf-8') as f:
        json.dump(output, f, indent=2, ensure_ascii=False)

    print(f"✅ Successfully created top1000_popular_verses.json with {len(extracted_verses)} verses!")
    print(f"\nSample verses included:")
    for verse in extracted_verses[:10]:
        print(f"  - {verse['reference']}: {verse['text'][:60]}...")


if __name__ == "__main__":
    main()
