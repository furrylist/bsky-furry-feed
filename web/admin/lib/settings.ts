const SHOW_QUEUE_ACTION_CONFIRMATION = "bff-show-queue-action-confirmation";
const BLUR_NSFW_POST_MEDIA = "bff-blur-nsfw-post-media";

function defineLocalStorageRef(key: string): Ref<boolean> {
  const r = useState(key, () => localStorage.getItem(key) === "true");
  watch(r, () => {
    console.log(key);
    if (r.value) {
      localStorage.setItem(key, "true");
    } else {
      localStorage.removeItem(key);
    }
  });
  return r;
}

export const showQueueActionConfirmation = defineLocalStorageRef(
  SHOW_QUEUE_ACTION_CONFIRMATION
);
export const blurNsfwPostMedia = defineLocalStorageRef(BLUR_NSFW_POST_MEDIA);
