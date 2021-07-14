import * as GQL from "src/core/generated-graphql";
import { blobToBase64 } from "base64-blob";
import { uniq } from "lodash";
import {
  useCreatePerformer,
  useCreateStudio,
  useCreateTag,
  useUpdatePerformerStashID,
  useUpdateStudioStashID,
} from "./queries";
import { IPerformerOperations } from "./PerformerResult";
import { StudioOperation } from "./StudioResult";
import { IStashBoxScene } from "./utils";

export interface ITagSceneOptions {
  setCoverImage?: boolean;
  setTags?: boolean;
  tagOperation: string;
}

export function useTagScene(
  options: ITagSceneOptions,
  setSaveState: (state: string) => void,
  setError: (err: { message?: string; details?: string }) => void
) {
  const createStudio = useCreateStudio();
  const createPerformer = useCreatePerformer();
  const createTag = useCreateTag();
  const updatePerformerStashID = useUpdatePerformerStashID();
  const updateStudioStashID = useUpdateStudioStashID();
  const [updateScene] = GQL.useSceneUpdateMutation({
    onError: (e) => {
      const message =
        e.message === "invalid JPEG format: short Huffman data"
          ? "Failed to save scene due to corrupted cover image"
          : "Failed to save scene";
      setError({
        message,
        details: e.message,
      });
    },
  });

  const { data: allTags } = GQL.useAllTagsForFilterQuery();

  const handleSave = async (
    stashScene: GQL.SlimSceneDataFragment,
    scene: IStashBoxScene,
    studio: StudioOperation | undefined,
    performers: IPerformerOperations,
    endpoint: string
  ) => {
    setError({});
    let performerIDs = [];
    let studioID = null;

    if (!studio) return;

    if (studio.type === "create") {
      setSaveState("Creating studio");
      const newStudio = {
        name: studio.data.name,
        stash_ids: [
          {
            endpoint,
            stash_id: scene.studio.stash_id,
          },
        ],
        url: studio.data.url,
      };
      const studioCreateResult = await createStudio(
        newStudio,
        scene.studio.stash_id
      );

      if (!studioCreateResult?.data?.studioCreate) {
        setError({
          message: `Failed to save studio "${newStudio.name}"`,
          details: studioCreateResult?.errors?.[0].message,
        });
        return setSaveState("");
      }
      studioID = studioCreateResult.data.studioCreate.id;
    } else if (studio.type === "update") {
      setSaveState("Saving studio stashID");
      const res = await updateStudioStashID(studio.data, [
        ...studio.data.stash_ids,
        { stash_id: scene.studio.stash_id, endpoint },
      ]);
      if (!res?.data?.studioUpdate) {
        setError({
          message: `Failed to save stashID to studio "${studio.data.name}"`,
          details: res?.errors?.[0].message,
        });
        return setSaveState("");
      }
      studioID = res.data.studioUpdate.id;
    } else if (studio.type === "existing") {
      studioID = studio.data.id;
    } else if (studio.type === "skip") {
      studioID = stashScene.studio?.id;
    }

    setSaveState("Saving performers");
    performerIDs = await Promise.all(
      Object.keys(performers).map(async (stashID) => {
        const performer = performers[stashID];
        if (performer.type === "skip") return "Skip";

        let performerID = performer.data.id;

        if (performer.type === "create") {
          const imgurl = performer.data.images[0];
          let imgData = null;
          if (imgurl) {
            const img = await fetch(imgurl, {
              mode: "cors",
              cache: "no-store",
            });
            if (img.status === 200) {
              const blob = await img.blob();
              imgData = await blobToBase64(blob);
            }
          }

          const performerInput = {
            name: performer.data.name,
            gender: performer.data.gender,
            country: performer.data.country,
            height: performer.data.height,
            ethnicity: performer.data.ethnicity,
            birthdate: performer.data.birthdate,
            eye_color: performer.data.eye_color,
            fake_tits: performer.data.fake_tits,
            measurements: performer.data.measurements,
            career_length: performer.data.career_length,
            tattoos: performer.data.tattoos,
            piercings: performer.data.piercings,
            twitter: performer.data.twitter,
            instagram: performer.data.instagram,
            image: imgData,
            stash_ids: [
              {
                endpoint,
                stash_id: stashID,
              },
            ],
            details: performer.data.details,
            death_date: performer.data.death_date,
            hair_color: performer.data.hair_color,
            weight: Number(performer.data.weight),
          };

          const res = await createPerformer(performerInput, stashID);
          if (!res?.data?.performerCreate) {
            setError({
              message: `Failed to save performer "${performerInput.name}"`,
              details: res?.errors?.[0].message,
            });
            return null;
          }
          performerID = res.data?.performerCreate.id;
        }

        if (performer.type === "update") {
          const stashIDs = performer.data.stash_ids;
          await updatePerformerStashID(performer.data.id, [
            ...stashIDs,
            { stash_id: stashID, endpoint },
          ]);
        }

        return performerID;
      })
    );

    if (!performerIDs.some((id) => !id)) {
      setSaveState("Updating scene");
      const imgurl = scene.images[0];
      let imgData = null;
      if (imgurl && options.setCoverImage) {
        const img = await fetch(imgurl, {
          mode: "cors",
          cache: "no-store",
        });
        if (img.status === 200) {
          const blob = await img.blob();
          // Sanity check on image size since bad images will fail
          if (blob.size > 10000) imgData = await blobToBase64(blob);
        }
      }

      let updatedTags = stashScene?.tags?.map((t) => t.id) ?? [];
      if (options.setTags) {
        const newTagIDs = options.tagOperation === "merge" ? updatedTags : [];
        const tags = scene.tags ?? [];
        if (tags.length > 0) {
          const tagDict: Record<string, string> = (allTags?.allTags ?? [])
            .filter((t) => t.name)
            .reduce(
              (dict, t) => ({ ...dict, [t.name.toLowerCase()]: t.id }),
              {}
            );
          const newTags: string[] = [];
          tags.forEach((tag) => {
            if (tagDict[tag.name.toLowerCase()])
              newTagIDs.push(tagDict[tag.name.toLowerCase()]);
            else newTags.push(tag.name);
          });

          const createdTags = await Promise.all(
            newTags.map((tag) => createTag(tag))
          );
          createdTags.forEach((createdTag) => {
            if (createdTag?.data?.tagCreate?.id)
              newTagIDs.push(createdTag.data.tagCreate.id);
          });
        }
        updatedTags = uniq(newTagIDs);
      }

      const performer_ids = performerIDs.filter(
        (id) => id !== "Skip"
      ) as string[];

      const sceneUpdateResult = await updateScene({
        variables: {
          input: {
            id: stashScene.id ?? "",
            title: scene.title,
            details: scene.details,
            date: scene.date,
            performer_ids:
              performer_ids.length === 0
                ? stashScene.performers.map((p) => p.id)
                : performer_ids,
            studio_id: studioID,
            cover_image: imgData,
            url: scene.url,
            tag_ids: updatedTags,
            stash_ids: [
              ...(stashScene?.stash_ids ?? []),
              {
                endpoint,
                stash_id: scene.stash_id,
              },
            ],
          },
        },
      });

      setSaveState("");
      return sceneUpdateResult?.data?.sceneUpdate;
    }

    setSaveState("");
  };

  return handleSave;
}
