<script lang="ts">
  export let data;
  $: title = data.feed?.displayName
    .replace('🐾', '')
    .replace(/[0-9]+$/, '')
    .trim();
  $: firstTag = data.feed?.description.match(/#\S+/)?.toString().toUpperCase();
</script>

{#if title}
  <div
    class="mx-auto my-3 flex w-full max-w-xl flex-col gap-3 border-4 border-transparent py-2 pb-6"
  >
    <h1 class="text-2xl font-bold">Welcome to Furrylist! 🐾</h1>
    <p>
      We're a group of volunteers that operates a bunch of different custom feeds, which
      serve up furry posts straight to your Bluesky app.
    </p>
    <div class="onboarding-card flex flex-col gap-2 rounded-md px-4 py-4">
      <h2 class="flex items-center gap-2 text-lg font-bold">
        <span
          class="onboarding-card flex h-6 w-6 shrink-0 select-none items-center justify-center rounded-full bg-gray-200 text-center text-sm dark:bg-gray-800"
          >1</span
        >
        Pin the {title} feed
      </h2>
      <div class="text-gray-700 dark:text-gray-400">
        Anyone can view our feeds. You can add the <span class="font-semibold"
          >{title}</span
        >
        feed to your Bluesky client by opening the feed and clicking the
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 256 256"
          class="mb-1 inline"
          ><!-- Icon from Phosphor by Phosphor Icons - https://github.com/phosphor-icons/core/blob/main/LICENSE --><path
            fill="currentColor"
            d="M216 164h-5.93L190.3 52h1.7a12 12 0 0 0 0-24H64a12 12 0 0 0 0 24h1.7L45.93 164H40a12 12 0 0 0 0 24h76v52a12 12 0 0 0 24 0v-52h76a12 12 0 0 0 0-24M90.07 52h75.86l19.77 112H70.3Z"
          /></svg
        > pin icon.
      </div>
      <a
        href={data.feed?.link}
        target="_blank"
        class="bg-blue ml-auto rounded-md bg-blue-700 px-2.5 py-1 text-white hover:bg-blue-600"
        >Pin the feed</a
      >
    </div>
    <div class="onboarding-card flex flex-col gap-2 rounded-md px-4 py-4">
      <h2 class="flex items-center gap-2 text-lg font-bold">
        <span
          class="onboarding-card flex h-6 w-6 shrink-0 select-none items-center justify-center rounded-full bg-gray-200 text-center text-sm dark:bg-gray-800"
          >2</span
        >
        Follow @furryli.st
      </h2>
      <div class="text-gray-700 dark:text-gray-400">
        If you are a furry (or pup, or therian, or brony), you can follow <a
          class="text-blue-700 underline hover:no-underline dark:text-blue-500"
          href="https://bsky.app/profile/furryli.st"
          target="_blank"
        >
          @furryli.st
        </a>
        to get your posts on the feeds. Our team reviews your account and follows you back soon
        if you align with our
        <a
          class="text-blue-700 underline hover:no-underline dark:text-blue-500"
          href="/community-guidelines">community guidelines</a
        >.
      </div>
      <a
        href="https://bsky.app/profile/furryli.st"
        target="_blank"
        class="bg-blue ml-auto rounded-md bg-blue-700 px-2.5 py-1 text-white hover:bg-blue-600"
        >Follow @furryli.st</a
      >
    </div>
    <div class="onboarding-card flex flex-col gap-2 rounded-md px-4 py-4">
      <h2 class="flex items-center gap-2 text-lg font-bold">
        <span
          class="onboarding-card flex h-6 w-6 shrink-0 select-none items-center justify-center rounded-full bg-gray-200 text-center text-sm dark:bg-gray-800"
          >3</span
        >
        Tag your posts
      </h2>
      <div class="text-gray-700 dark:text-gray-400">
        Now it's time to tag your posts with {firstTag} and join the conversation about {title}.
      </div>
      <a
        href="https://bsky.app/"
        target="_blank"
        class="bg-blue ml-auto rounded-md bg-blue-700 px-2.5 py-1 text-white hover:bg-blue-600"
        >Let's go posting</a
      >
    </div>
    <p>
      Curious about other feeds? Check out <a
        href="/feeds"
        class="text-blue-700 underline hover:no-underline dark:text-blue-500"
        >all our feeds</a
      >!
    </p>
    <p>
      Want to learn more? Read our <a
        href="/welcome"
        class="text-blue-700 underline hover:no-underline dark:text-blue-500"
        >full welcome guide</a
      >!
    </p>
  </div>
{/if}

<style lang="scss">
  .onboarding-card {
    @apply my-0.5 from-gray-300/5;
    box-shadow: 0 0 0 2px var(--tw-gradient-from);
  }

  :global(.dark) {
    .onboarding-card {
      @apply from-gray-500/50;
      box-shadow: 0 0 0 2px var(--tw-gradient-from);
    }
  }
</style>
