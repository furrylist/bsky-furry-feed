<script setup lang="ts">
import { PostView } from "@atproto/api/dist/client/types/app/bsky/feed/defs";
import { newAgent } from "~/lib/auth";
import { chunk } from "~/lib/util";
import { SignJWT, generateKeyPair } from "jose";

const agent = newAgent();
const currentUser = await useUser();
const publicApi = await usePublicAPI();
const { feeds } = await publicApi.listFeeds({});

const currentFeedId = ref("furry-test");

const cursor = ref("");
const posts = ref<PostView[]>([]);
const loading = ref(false);

// generateJwt generates a mock JWT that just sets the DID as issuer.
async function generateJwt() {
  const { privateKey } = await generateKeyPair("PS256");
  return await new SignJWT({})
    .setIssuer(currentUser.value.did)
    .setProtectedHeader({ alg: "PS256" })
    .sign(privateKey);
}

async function urisToPosts(allUris: string[]) {
  const allPosts: PostView[] = [];
  for (const uris of chunk(allUris, 25)) {
    const { data } = await agent.getPosts({ uris });

    allPosts.push(...data.posts);
  }
  return allPosts;
}

async function fetchFeedSkeleton() {
  loading.value = true;
  const { apiUrl } = useRuntimeConfig().public;
  const url = new URL(apiUrl);
  url.pathname = "/xrpc/app.bsky.feed.getFeedSkeleton";
  url.searchParams.set("feed", currentFeedId.value);
  if (cursor.value) {
    url.searchParams.set("cursor", cursor.value);
  }

  const resp: any = await $fetch(url.href, {
    headers: { authorization: `Bearer ${await generateJwt()}` },
  });
  const postURIs = resp.feed
    ?.map((p: any) => p.post)
    .filter(Boolean) as string[];
  cursor.value = resp.cursor;

  posts.value = [...posts.value, ...(await urisToPosts(postURIs))];
  loading.value = false;
}

onMounted(() => {
  watch(
    currentFeedId,
    async () => {
      posts.value = [];
      await fetchFeedSkeleton();
    },
    { immediate: true }
  );
});
</script>

<template>
  <div class="mb-3">
    <label for="feed" class="text-muted block mb-0.5 text-sm">Feed</label>
    <select
      class="dark:text-black p-1 px-2 rounded"
      name="feed"
      id="feed"
      v-model="currentFeedId"
    >
      <option
        v-for="feed in feeds.sort((a, b) => a.id.localeCompare(b.id))"
        :key="feed.id"
        :value="feed.id"
      >
        {{ feed.displayName }} ({{ feed.id }})
      </option>
    </select>
  </div>
  <div
    class="md:max-w-[80%] border border-gray-300 dark:border-gray-700 rounded-lg mb-3"
  >
    <UserRecentPost v-for="post in posts" :post="{ post }" />
  </div>
  <div class="flex justify-center md:max-w-[80%]">
    <button
      class="py-1 max-md:py-1.5 max-md:px-3 px-2 max-md:ml-auto mr-1 text-white bg-blue-500 dark:bg-blue-600 rounded-lg hover:bg-blue-600 dark:hover:bg-blue-700 disabled:bg-blue-300 disabled:dark:bg-blue-500 disabled:cursor-not-allowed"
      :disabled="loading"
      @click="fetchFeedSkeleton"
    >
      Load more
    </button>
  </div>
</template>
