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

const posts = ref<PostView[]>([]);

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
  const { apiUrl } = useRuntimeConfig().public;
  const url = new URL(apiUrl);
  url.pathname = "/xrpc/app.bsky.feed.getFeedSkeleton";
  url.searchParams.set("feed", currentFeedId.value);

  const postURIs = await $fetch(url.href, {
    headers: { authorization: `Bearer ${await generateJwt()}` },
  }).then(
    (r: any) => r.feed?.map((p: any) => p.post).filter(Boolean) as string[]
  );

  posts.value = await urisToPosts(postURIs);
}

onMounted(() => {
  watch(currentFeedId, fetchFeedSkeleton, { immediate: true });
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
    class="max-w-[80%] border border-gray-300 dark:border-gray-700 rounded-lg"
  >
    <UserRecentPost v-for="post in posts" :post="{ post }" />
  </div>
</template>
