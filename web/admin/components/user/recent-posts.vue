<script setup lang="ts">
import { AppBskyFeedDefs } from "@atproto/api";
import { hasFurryHashtag } from "~/lib/furry-detector";

const props = defineProps<{
  actorDid: string;
  posts: Array<AppBskyFeedDefs.FeedViewPost>;
}>();

const hasFurryTags = computed(() =>
  props.posts.map((p) => String((p.record as any)?.text)).some(hasFurryHashtag)
);
</script>

<template>
  <div
    class="px-4 py-3 border-b border-gray-300 dark:border-gray-700 flex items-center"
  >
    <h2>Recent posts</h2>
    <span
      v-if="hasFurryTags"
      class="ml-auto text-sm bg-teal-700 flex items-center gap-0.5 px-1 rounded-lg h-min"
    >
      <icon-check />
      furry tags
    </span>
  </div>
  <div class="overflow-y-auto max-h-[500px]">
    <template v-for="post in posts" :key="post.post.uri">
      <user-recent-post v-if="post" :actor-did="actorDid" :post="post" />
    </template>
    <div
      v-if="posts.length === 0"
      class="text-muted px-4 py-2 border-gray-300 dark:border-gray-700"
    >
      No recent posts.
    </div>
  </div>
</template>
