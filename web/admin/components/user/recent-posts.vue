<script setup lang="ts">
import { AppBskyFeedDefs } from "@atproto/api";
import { hasFurryHashtag } from "~/lib/furry-detector";

const props = defineProps<{
  actorDid: string;
  posts: Array<AppBskyFeedDefs.FeedViewPost>;
}>();

const showReposts = useState("user-recent-posts_show-reposts", () => true);

const hasFurryTags = computed(() =>
  props.posts
    .filter((p) => p.post.author.did === props.actorDid)
    .map((p) => String((p.post.record as any)?.text))
    .some(hasFurryHashtag)
);

const filteredPosts = computed(() => {
  if (showReposts.value) {
    return props.posts;
  }

  return props.posts.filter(
    (post) =>
      post.reason?.$type !== "app.bsky.feed.defs#reasonRepost" ||
      post.post.author.did === props.actorDid
  );
});
</script>

<template>
  <div
    class="px-4 py-3 border-b border-gray-300 dark:border-gray-700 flex items-center"
  >
    <h2>
      Recent posts
      <span class="text-muted"
        >(<button
          class="underline hover:no-underline"
          @click="showReposts = !showReposts"
        >
          {{ showReposts ? "with reposts" : "no reposts" }}</button
        >)</span
      >
    </h2>
    <span
      v-if="hasFurryTags"
      class="ml-auto text-sm bg-teal-700 flex items-center gap-0.5 px-1 rounded-lg h-min"
    >
      <icon-check />
      furry tags
    </span>
  </div>
  <div class="overflow-y-auto max-h-[500px]">
    <template v-for="post in filteredPosts" :key="post.post.uri">
      <user-recent-post v-if="post" :actor-did="actorDid" :post="post" />
    </template>
    <div
      v-if="filteredPosts.length === 0"
      class="text-muted px-4 py-2 border-gray-300 dark:border-gray-700"
    >
      No recent posts.
    </div>
  </div>
</template>
