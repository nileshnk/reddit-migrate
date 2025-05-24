# Reddit Migration Test Scripts

This directory contains utility scripts for testing the Reddit migration functionality.

## Reddit Test Data Generator

The `reddit-test-data.js` script allows you to automatically follow random non-NSFW subreddits and save random non-NSFW posts to generate test data for your Reddit migration workflows. It also provides cleanup operations to unfollow all subreddits and unsave all posts.

### Prerequisites

- Node.js (version 12 or higher)
- A valid Reddit cookie from your browser

### Getting Your Reddit Cookie

1. Open your browser and log into Reddit
2. Open Developer Tools (F12)
3. Go to Application/Storage → Cookies → reddit.com
4. Copy the cookie string (all values separated by semicolons)

### Usage

#### Basic Usage

```bash
# Follow 10 subreddits and save 10 posts (defaults)
node scripts/reddit-test-data.js

# Custom amounts
node scripts/reddit-test-data.js --subreddits 50 --posts 100

# Only follow subreddits (no posts)
node scripts/reddit-test-data.js --subreddits 25 --posts 0

# Only save posts (no subreddits)
node scripts/reddit-test-data.js --subreddits 0 --posts 50
```

#### Cleanup Operations

```bash
# Unfollow all currently followed subreddits
node scripts/reddit-test-data.js --unfollowAll --cookie "your_cookie_here"

# Unsave all currently saved posts
node scripts/reddit-test-data.js --unsaveAll --cookie "your_cookie_here"

# Do both cleanup operations
node scripts/reddit-test-data.js --unfollowAll --unsaveAll --cookie "your_cookie_here"
```

#### Using Cookie Argument

```bash
node scripts/reddit-test-data.js --subreddits 100 --posts 100 --cookie "reddit_session=abc123;token_v2=xyz789"
```

#### Or Edit the Script

You can paste your cookie directly in the script by editing the `DEFAULT_COOKIE` constant:

```javascript
const DEFAULT_COOKIE = `
reddit_session=your_actual_session_here;
token_v2=your_actual_token_here;
csv=your_actual_csv_here;
session_tracker=your_actual_tracker_here;
`
  .replace(/\s+/g, "")
  .replace(/;$/, "");
```

### Command Line Options

| Option                  | Description                                | Default             |
| ----------------------- | ------------------------------------------ | ------------------- |
| `--subreddits <number>` | Number of subreddits to follow             | 10                  |
| `--posts <number>`      | Number of posts to save                    | 10                  |
| `--cookie <string>`     | Reddit cookie string                       | Uses DEFAULT_COOKIE |
| `--unfollowAll`         | Unfollow all currently followed subreddits | false               |
| `--unsaveAll`           | Unsave all currently saved posts           | false               |
| `--help`, `-h`          | Show help message                          | -                   |

### Features

- ✅ **Safe**: Only processes non-NSFW content
- ✅ **Random Selection**: Uses Fisher-Yates shuffle algorithm for truly random selection
- ✅ **Bulk Operations**: Efficiently follows multiple subreddits in chunks for better performance
- ✅ **Rate Limit Handling**: Automatically detects and stops on Reddit's 429 rate limit errors
- ✅ **Smart Delays**: Includes configurable delays between requests to avoid hitting limits
- ✅ **Progress Tracking**: Shows real-time progress with percentage and statistics
- ✅ **Error Recovery**: Graceful error handling with detailed reporting
- ✅ **Authentication Check**: Validates your cookie before starting any operations
- ✅ **Cleanup Operations**: Can unfollow all subreddits and unsave all posts for clean testing
- ✅ **Detailed Logging**: Shows exactly what's being followed/saved with subscriber counts
- ✅ **Partial Success**: Continues processing even if some operations fail
- ✅ **OAuth Support**: Automatically extracts and uses Bearer tokens when available

### Rate Limiting

The script includes advanced rate limit handling:

- **Automatic Detection**: Detects HTTP 429 (Too Many Requests) responses
- **Immediate Stop**: Stops all operations when rate limit is hit
- **Clear Messaging**: Provides guidance on how long to wait (10-15 minutes)
- **Progress Preservation**: Reports how many operations completed before hitting the limit
- **Built-in Delays**: 1-second delays between individual operations to minimize risk

If you hit a rate limit, you'll see:

```
⚠️  Rate limit hit (429) - Reddit is limiting requests
🕐 Please wait at least 10-15 minutes before trying again
💡 Consider reducing the number of items or spreading requests over time
```

### Example Output

```bash
🚀 Reddit Test Data Generator
===============================
Operations: follow 25 subreddits, save 50 posts

🔐 Validating authentication...
✅ Authenticated as: your_username

🎯 Following 25 random subreddits...
🔍 Fetching 75 popular subreddits...
   Fetching batch: 1-75
✅ Fetched 75 total subreddits
📊 Fetched 75 subreddits for randomization
📋 Selected 25 subreddits to follow:
   1. r/programming (2,500,000 subscribers)
   2. r/science (25,000,000 subscribers)
   ...

✅ Successfully followed: r/programming, r/science, r/technology...
Progress: 100% (25/25)

💾 Saving 50 random posts...
🔍 Fetching 100 posts from r/all...
   Fetching batch: 1-100
✅ Fetched 100 total posts
📊 Fetched 100 posts for randomization
📋 Selected 50 posts to save:
   1. "Amazing discovery in quantum computing" from r/science
   2. "New JavaScript framework released" from r/programming
   ...

Saving "Amazing discovery in quantum..."... ✅
Progress: 2% (1/50)
...

📊 Summary:
===========
✅ Subreddits followed: 25/25
✅ Posts saved: 50/50
❌ Errors: 0
⏱️  Duration: 180 seconds

🎉 Operation complete!
You can now test your migration script with this data.
```

### Cleanup Example

```bash
🚀 Reddit Test Data Generator
===============================
Operations: unfollow all subreddits, unsave all posts

🔐 Validating authentication...
✅ Authenticated as: your_username

🎯 Unfollowing all subreddits...
🔍 Fetching followed subreddits...
📋 Found 25 subreddits to unfollow:
   1. r/programming
   2. r/science
   ...

Unfollowing r/programming... ✅
Progress: 4% (1/25)
...

💾 Unsaving all posts...
🔍 Fetching saved posts...
📋 Found 50 posts to unsave:
   1. "Amazing discovery in quantum computing" from r/science
   2. "New JavaScript framework released" from r/programming
   ...

📊 Summary:
===========
✅ Subreddits unfollowed: 25
✅ Posts unsaved: 50
❌ Errors: 0
⏱️  Duration: 120 seconds

🎉 Operation complete!
Cleanup operations finished.
```

### Testing Your Migration

After running this script, you'll have:

- Random subreddits in your "old" account's subscriptions
- Random saved posts in your "old" account

You can then test your migration tool by:

1. Using this account as the "old" account
2. Creating or using another account as the "new" account
3. Running your migration to transfer the generated data

For cleanup after testing:

1. Run with `--unfollowAll` to remove all test subreddits
2. Run with `--unsaveAll` to remove all test saved posts
3. Or combine both flags to do a complete cleanup

### Troubleshooting

#### Authentication Issues

- Make sure your cookie is complete and current
- Try logging out and back into Reddit to get a fresh cookie
- Check that all required cookie components are included
- Ensure the token_v2 value is present for OAuth authentication

#### Rate Limiting

- The script includes automatic rate limit detection
- If you hit rate limits (429 error), wait 10-15 minutes before retrying
- Consider reducing the numbers or running during off-peak hours
- Reddit has stricter limits during peak hours (US daytime)
- Use smaller numbers for testing (e.g., 10-25 items)

#### Network Issues

- Ensure you have a stable internet connection
- Some corporate networks may block Reddit API calls
- Try running from a different network if issues persist
- Check if Reddit is experiencing downtime

### Safety Notes

- This script only reads from and writes to Reddit APIs
- It never deletes or modifies existing data (except when using cleanup flags)
- All operations are reversible through Reddit's interface or cleanup commands
- Only non-NSFW content is processed
- Respects Reddit's rate limiting guidelines with built-in delays
- Stops immediately when rate limits are detected

### Advanced Usage

#### Combining Operations

You can combine different operations based on your testing needs:

```bash
# Generate test data with specific amounts
node scripts/reddit-test-data.js --subreddits 100 --posts 200

# Clean up everything after testing
node scripts/reddit-test-data.js --unfollowAll --unsaveAll

# Only clean up subreddits but keep saved posts
node scripts/reddit-test-data.js --unfollowAll
```

#### Best Practices

1. **Start Small**: Test with small numbers first (10-25 items)
2. **Monitor Progress**: Watch the console output for any issues
3. **Respect Limits**: Don't run the script repeatedly in quick succession
4. **Use Cleanup**: Always clean up test data when done testing
5. **Fresh Cookie**: Get a new cookie if you encounter persistent auth issues

### Integration with Migration Tool

This script generates the perfect test data for your migration tool because:

- It creates real Reddit subscriptions and saved posts
- The data structure matches exactly what your migration tool expects
- You can control the exact amount of test data
- All content is safe and non-controversial
- Cleanup operations ensure you can reset to a clean state
- Rate limit handling prevents account issues during testing
