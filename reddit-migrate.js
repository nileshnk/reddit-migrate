const OLD_ACC_TOKEN = ``;
const NEW_ACC_TOKEN = ``;

// Migrate subreddits
async function subredditMigrate(
  OLD_ACC_TOKEN,
  NEW_ACC_TOKEN,
  UNSUB_OLD_ACC = false
) {
  // fetch usernames
  const oldUsername = await fetchUsername(OLD_ACC_TOKEN);
  const newUsername = await fetchUsername(NEW_ACC_TOKEN);
  // fetch all subreddits from old account
  console.log(`Fetching subreddits from ${oldUsername} account...`);
  const subredditFullNameList = await fetchSubredditFullNames(OLD_ACC_TOKEN);
  console.log(
    `Total subreddits in ${oldUsername} account:`,
    subredditFullNameList.length
  );
  // subscribe to all subreddits from new account
  console.log(
    `Subscribing to ${subredditFullNameList.length} subreddits from ${newUsername} account...`
  );
  const subscribeData = await manageSubreddits(
    NEW_ACC_TOKEN,
    subredditFullNameList,
    SUBSCRIBE_TYPE.SUBSCRIBE
  );
  if (subscribeData.error) {
    return;
  }
  console.log(
    `Subscribed to ${subredditFullNameList.length} subreddits from ${newUsername} account`
  );

  if (UNSUB_OLD_ACC === true) {
    console.log(
      `Unsubscribing ${subredditFullNameList.length} subreddits from ${oldUsername} account...`
    );
    // Unsubscribe from subreddits
    await manageSubreddits(
      OLD_ACC_TOKEN,
      subredditFullNameList,
      SUBSCRIBE_TYPE.UNSUBSCRIBE
    );
    console.log(
      `Unsubscribed ${subredditFullNameList.length} subreddits from ${oldUsername} account`
    );
  }
}

// Migrate saved posts

async function savedPostsMigrate(
  OLD_ACC_TOKEN,
  NEW_ACC_TOKEN,
  UNSAVE_OLD_ACC = false
) {
  const oldUsername = await fetchUsername(OLD_ACC_TOKEN);
  const newUsername = await fetchUsername(NEW_ACC_TOKEN);
  // fetch all saved posts from old account
  console.log(`Fetching saved posts from ${oldUsername} account...`);
  const savedPostsFullNames = await fetchSavedPostFullNames(OLD_ACC_TOKEN);
  console.log(
    `Total saved posts in ${oldUsername} account:`,
    savedPostsFullNames.length
  );
  // save all posts to new account
  console.log(
    `Saving ${savedPostsFullNames.length} posts to ${newUsername} account...`
  );
  const savePostsResponse = await managePosts(
    NEW_ACC_TOKEN,
    savedPostsFullNames,
    SAVE_TYPE.SAVE
  );
  console.log(
    `Saved ${savePostsResponse.successCount} posts to ${newUsername} account`
  );
  // unsave all posts from old account
  if (UNSAVE_OLD_ACC === true) {
    console.log(`Unsaving posts from ${oldUsername} account...`);
    const unsavePostsResponse = await managePosts(
      OLD_ACC_TOKEN,
      savedPostsFullNames,
      SAVE_TYPE.UNSAVE
    );
    console.log(
      `Unsaved ${unsavePostsResponse.successCount} posts from ${oldUsername} account`
    );
  }
}

const SUBSCRIBE_TYPE = {
  SUBSCRIBE: "sub",
  UNSUBSCRIBE: "unsub",
};

const SAVE_TYPE = {
  SAVE: "save",
  UNSAVE: "unsave",
};

const fetchUsername = async (token) => {
  // fetch username
  const fetchUserName = await fetch("https://oauth.reddit.com/api/me.json", {
    headers: {
      Authorization: token,
    },
  });
  const userNameData = await fetchUserName.json();
  return userNameData.data.name;
};

async function fetchSavedPostFullNames(token) {
  const username = await fetchUsername(token);
  // fetch saved posts
  const requiredAPI = `https://oauth.reddit.com/user/${username}/saved.json`;
  const savedPostsFullNames = await fetchAllFullNames(token, requiredAPI);
  return savedPostsFullNames;
}

const fetchSubredditFullNames = async (token) => {
  const requiredAPI = "https://oauth.reddit.com/subreddits/mine.json";
  const fullNameList = await fetchAllFullNames(token, requiredAPI);
  return fullNameList;
};

async function fetchAllFullNames(token, requiredAPI) {
  const fullNameList = [];
  const display_name = [];
  let lastFullName = "";
  while (true) {
    const response = await fetch(
      `${requiredAPI}?limit=100&after=${lastFullName}`,
      {
        headers: {
          Authorization: token,
        },
        body: null,
        method: "GET",
      }
    );
    const json = await response.json();

    const names = json.data.children.map((child) => child.data.name);
    display_name.push(
      json.data.children.map((child) => {
        return { name: child.data.name, display_name: child.data.display_name };
      })
    );

    if (names.length === 100) {
      lastFullName = names[99];
    } else if (names.length < 100) {
      fullNameList.push(...names);
      break;
    }
    fullNameList.push(...names);
  }
  // TODO: for future use
  // const name_displayname_list = display_name.flat();
  // const name_displayname_map = new Map();
  // // name_displayname_list.forEach((item) => {
  // //   name_displayname_map.set(item.name, item.display_name);
  // // });
  return fullNameList;
}

function splitArrayIntoChunks(array, chunkSize) {
  const chunkedArray = [];
  for (let i = 0; i < array.length; i += chunkSize) {
    chunkedArray.push(array.slice(i, i + chunkSize));
  }
  return chunkedArray;
}

async function manageSubreddits(
  token,
  subredditFullNameList,
  subscribeType = SUBSCRIBE_TYPE.SUBSCRIBE
) {
  const ChunkSize = 200;
  const subredditFullNameListChunks = splitArrayIntoChunks(
    subredditFullNameList,
    ChunkSize
  );

  const response = await Promise.all(
    subredditFullNameListChunks.map(async (subredditFullNameChunk) => {
      const subredditNames = subredditFullNameChunk.join(",");
      const response = await fetch("https://oauth.reddit.com/api/subscribe", {
        headers: {
          Authorization: token,
          "content-type": "application/x-www-form-urlencoded; charset=UTF-8",
          accept: "application/json, text/javascript, */*; q=0.01",
        },
        // body: `action=${subscribeType}&sr_name=${subredditNames}&api_type=json`,
        body: `sr=${subredditNames}&action=${subscribeType}&api_type=json`,
        method: "POST",
      });
      console.log(response.status, await response.json());
      if (response.status !== 200) {
        console.log(response.status, await response.json());
        return {
          error: true,
          status: response.status,
          successCount: 0,
          failedCount: subredditFullNameChunk.length,
          failedSubreddits: subredditFullNameChunk,
        };
      } else {
        // const json = await response.json();
        return {
          error: false,
          status: response.status,
          successCount: subredditFullNameChunk.length,
          failedCount: 0,
          failedSubreddits: [],
        };
      }
    })
  );
  const FinalResponse = {};
  response.forEach((res) => {
    if (res.error === true) {
      console.log(res.status, res.failedSubreddits);
      FinalResponse.error = true;
      FinalResponse.status = res.status;
      FinalResponse.failedCount += res.failedCount;
      FinalResponse.failedSubreddits.push(...res.failedSubreddits);
    }
    FinalResponse.successCount += res.successCount;
  });
  console.log(FinalResponse);
  return FinalResponse;
}

async function managePosts(token, postIds, saveType = SAVE_TYPE.SAVE) {
  let failedSavePostIds = [];
  postIds.forEach(async (postId) => {
    const response = await fetch(`https://oauth.reddit.com/api/${saveType}`, {
      headers: {
        Authorization: token,
        "Content-Type": "application/x-www-form-urlencoded",
      },
      // body: `id=${postId}&uh=${modhash}`,
      body: `id=${postId}`,
      method: "POST",
    });

    if (response.status !== 200) {
      const data = {
        fullname: postId,
        status: response.status,
        data: await response.json(),
      };
      failedSavePostIds.push(data);
    }
    // ++count;
    // console.log(count, await response.json());
    // if (count === 100) {
    //   // wait for 1 minute
    //   console.log("waiting for 1 minute.. as 100 posts are saved");
    //   await new Promise((resolve) => setTimeout(resolve, 60000));
    //   count = 0;
    // }
  });
  if (failedSavePostIds.length !== 0) {
    console.log(failedSavePostIds);
  }
  return {
    successCount: postIds.length - failedSavePostIds.length,
    failedCount: failedSavePostIds,
  };
}

savedPostsMigrate(OLD_ACC_TOKEN, NEW_ACC_TOKEN, false);
subredditMigrate(OLD_ACC_TOKEN, NEW_ACC_TOKEN, false);
