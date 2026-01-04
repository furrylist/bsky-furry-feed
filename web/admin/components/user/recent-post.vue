<script setup lang="ts">
import { blurNsfwPostMedia } from "~/lib/settings";
import { addSISuffix } from "~/lib/util";
import { ViewImage } from "@atproto/api/dist/client/types/app/bsky/embed/images";

import { AppBskyFeedDefs } from "@atproto/api";
import { ActorStatus } from "../../../proto/bff/v1/types_pb";

const props = defineProps<{
  actorDid: string;
  post: AppBskyFeedDefs.FeedViewPost;
}>();

const postType = computed(() => {
  if (props.post.reply) return "reply";
  if (props.post.reason?.$type === "app.bsky.feed.defs#reasonRepost")
    return props.post.post.author.did === props.actorDid
      ? "self-repost"
      : "repost";
  if (props.post.post.author.did !== props.actorDid) return "unknown-other";
  return "post";
});
const showPost = computed(
  () =>
    postType.value === "post" ||
    postType.value === "repost" ||
    postType.value === "self-repost"
);

const authorStatus = ref<ActorStatus>();

onMounted(async () => {
  if (postType.value === "repost") {
    const api = await useAPI();
    const resp = await api.getActor({ did: props.post.post.author.did });
    authorStatus.value = resp.actor?.status;
  }
});

const images = computed(() => {
  if (props.post.post.embed && "images" in props.post.post.embed) {
    const images = props.post.post.embed.images as (ViewImage & {
      type: "image";
    })[];
    images.forEach((img) => (img.type = "image"));
    return images;
  }
  if ((props.post.post.embed?.media as any)?.images) {
    const images = (props.post.post.embed?.media as any)
      .images as (ViewImage & {
      type: "image";
    })[];
    images.forEach((img) => (img.type = "image"));
    return images;
  }
  if (
    props.post.post.embed?.$type === "app.bsky.embed.video#view" &&
    props.post.post.embed.thumbnail
  ) {
    return [
      {
        alt: "",
        fullsize: props.post.post.embed.thumbnail,
        thumb: props.post.post.embed.thumbnail,
      },
    ] as (ViewImage & { type: "video" })[];
  }
  return [];
});

const labels = computed(() => props.post.post.labels || []);

const isNSFW = computed(() => {
  const values = labels.value.map((l) => l.val);
  return ["porn", "nudity", "sexual"].some((val) => values.includes(val));
});
</script>

<template>
  <div
    v-if="showPost"
    class="border-b border-gray-300 dark:border-gray-700"
    :class="postType === 'repost' ? 'bg-gray-200/40 dark:bg-gray-950/40' : ''"
  >
    <div class="px-4 py-2">
      <div
        v-if="postType === 'repost' || postType === 'self-repost'"
        class="text-muted opacity-80 w-full flex items-center gap-0.5 mb-0.5 text-xs"
      >
        <icon-reskeet class="h-4 w-4" />
        <span v-if="postType === 'self-repost'">Self-reposting</span>
        <span v-else
          >Reposting
          <nuxt-link
            class="underline hover:no-underline"
            :href="`/users/${post.post.author.did}`"
            >@{{ post.post.author.handle }}</nuxt-link
          >
          <user-status-badge
            v-if="authorStatus"
            class="ml-1"
            :status="authorStatus"
            tiny
          />
        </span>
      </div>
      <div class="meta text-sm text-muted">
        <span class="meta-item">
          <shared-date :date="new Date(post.post.indexedAt)" />
        </span>
        <span class="meta-item flex items-center gap-0.5">
          <icon-heart class="text-muted" />
          {{ addSISuffix(post.post.likeCount || 0) }}
        </span>
        <span class="meta-item flex items-center gap-0.5">
          <icon-square-bubble class="text-muted" :size="14" />
          {{ addSISuffix(post.post.replyCount || 0) }}
        </span>
      </div>
      <div class="flex">
        <shared-bsky-description
          :description="(post.post.record as any)?.text"
          class="flex-1"
        />

        <span
          v-if="images.length"
          class="w-[25%] h-100 flex-shrink-0 flex flex-col gap-1"
        >
          <template v-for="img in images" :key="img.thumb">
            <core-zoomable
              v-if="img.type === 'image'"
              :label="
                isNSFW && blurNsfwPostMedia
                  ? 'Click to zoom and unblur'
                  : undefined
              "
            >
              <img
                class="object-cover h-100 rounded-lg"
                :class="{ 'blur-md': isNSFW && blurNsfwPostMedia }"
                :src="img.thumb"
                :alt="img.alt"
              />
              <template #fullsize="{ classes }">
                <img
                  class="object-cover h-100"
                  :class="classes"
                  :src="img.fullsize"
                  :alt="img.alt"
                />
              </template>
            </core-zoomable>
            <nuxt-link
              v-else
              :href="`https://bsky.app/profile/${
                post.post.author.did
              }/post/${post.post.uri.split('/').pop()}`"
              class="hover:opacity-90 relative"
              title="View on Bluesky"
            >
              <div
                class="absolute w-full h-full flex items-center justify-center"
              >
                <div
                  class="p-0.5 bg-gray-900/50 flex items-center justify-center rounded-lg"
                >
                  <icon-play class="h-7 w-7" />
                </div>
              </div>
              <div class="overflow-hidden rounded-lg">
                <img
                  class="object-cover h-100"
                  :class="{ 'blur-md': isNSFW && blurNsfwPostMedia }"
                  :src="img.thumb"
                  :alt="img.alt"
                />
              </div>
            </nuxt-link>
          </template>
        </span>
      </div>
    </div>
    <div
      v-if="labels.length"
      class="text-xs text-muted px-4 py-1 bg-gray-100 dark:bg-gray-950 dark:bg-gray border-t border-dashed border-gray-300 dark:border-gray-700"
      style="padding: -2rem"
    >
      <div
        v-for="label in props.post.post.labels"
        :key="label.cid"
        class="flex items-center gap-0.5"
      >
        <div>
          Labeled <code>{{ label.val }}</code> by
        </div>
        <user-link :did="label.src" />
      </div>
    </div>
  </div>
</template>
