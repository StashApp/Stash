import React, { useState, useReducer } from "react";
import cx from "classnames";
import { Button } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { uniq } from "lodash";
import { blobToBase64 } from "base64-blob";
import distance from "hamming-distance";

import * as GQL from "src/core/generated-graphql";
import {
  LoadingIndicator,
  HoverPopover,
  SuccessIcon,
  TruncatedText,
} from "src/components/Shared";
import PerformerResult, { PerformerOperation } from "../PerformerResult";
import StudioResult, { StudioOperation } from "./StudioResult";
import { IStashBoxScene } from "../utils";
import {
  useCreateTag,
  useCreatePerformer,
  useCreateStudio,
  useUpdatePerformerStashID,
  useUpdateStudioStashID,
} from "../queries";

const getDurationStatus = (
  scene: IStashBoxScene,
  stashDuration: number | undefined | null
) => {
  if (!stashDuration) return "";

  const durations = scene.fingerprints
    .map((f) => f.duration)
    .map((d) => Math.abs(d - stashDuration));
  const matchCount = durations.filter((duration) => duration <= 5).length;

  let match;
  if (matchCount > 0)
    match = (
      <FormattedMessage
        id="component_tagger.results.fp_matches_multi"
        values={{ matchCount, durationsLength: durations.length }}
      />
    );
  else if (Math.abs(scene.duration - stashDuration) < 5)
    match = <FormattedMessage id="component_tagger.results.fp_matches" />;

  if (match)
    return (
      <div className="font-weight-bold">
        <SuccessIcon className="mr-2" />
        {match}
      </div>
    );

  const minDiff = Math.min(
    Math.abs(scene.duration - stashDuration),
    ...durations
  );
  return <div>Duration off by at least {Math.floor(minDiff)}s</div>;
};

const getFingerprintStatus = (
  scene: IStashBoxScene,
  stashScene: GQL.SlimSceneDataFragment
) => {
  const checksumMatch = scene.fingerprints.some(
    (f) => f.hash === stashScene.checksum || f.hash === stashScene.oshash
  );
  const phashMatches = scene.fingerprints.filter(
    (f) => f.algorithm === "PHASH" && distance(f.hash, stashScene.phash) <= 8
  );

  const phashList = (
    <div className="m-2">
      {phashMatches.map((fp) => (
        <div>
          <b>{fp.hash}</b>
          {fp.hash === stashScene.phash
            ? ", Exact match"
            : `, distance ${distance(fp.hash, stashScene.phash)}`}
        </div>
      ))}
    </div>
  );

  if (checksumMatch || phashMatches.length > 0)
    return (
      <div className="font-weight-bold">
        <SuccessIcon className="mr-2" />
        {phashMatches.length > 0 ? (
          <HoverPopover
            placement="bottom"
            content={phashList}
            className="PHashPopover"
          >
            {phashMatches.length > 1 ? (
              <FormattedMessage
                id="component_tagger.results.phash_matches"
                values={{
                  count: phashMatches.length,
                }}
              />
            ) : (
              <FormattedMessage
                id="component_tagger.results.hash_matches"
                values={{
                  hash_type: <FormattedMessage id="media_info.phash" />,
                }}
              />
            )}
          </HoverPopover>
        ) : (
          <FormattedMessage
            id="component_tagger.results.hash_matches"
            values={{
              hash_type: <FormattedMessage id="media_info.checksum" />,
            }}
          />
        )}
      </div>
    );
};

interface IStashSearchResultProps {
  scene: IStashBoxScene;
  stashScene: GQL.SlimSceneDataFragment;
  isActive: boolean;
  setActive: () => void;
  showMales: boolean;
  setScene: (scene: GQL.SlimSceneDataFragment) => void;
  excludedFields?: string[];
  tagOperation: string;
  setTags: boolean;
  endpoint: string;
  queueFingerprintSubmission: (sceneId: string, endpoint: string) => void;
}

interface IPerformerReducerAction {
  id: string;
  data: PerformerOperation;
}

const performerReducer = (
  state: Record<string, PerformerOperation>,
  action: IPerformerReducerAction
) => ({ ...state, [action.id]: action.data });

const StashSearchResult: React.FC<IStashSearchResultProps> = ({
  scene,
  stashScene,
  isActive,
  setActive,
  showMales,
  setScene,
  excludedFields = [],
  tagOperation,
  setTags,
  endpoint,
  queueFingerprintSubmission,
}) => {
  const [studio, setStudio] = useState<StudioOperation>();
  const [performers, dispatch] = useReducer(performerReducer, {});
  const [saveState, setSaveState] = useState<string>("");
  const [error, setError] = useState<{ message?: string; details?: string }>(
    {}
  );

  const intl = useIntl();
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

  const setPerformer = (
    performerData: PerformerOperation,
    performerID: string
  ) => dispatch({ id: performerID, data: performerData });

  const handleSave = async () => {
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
      if (imgurl && !excludedFields.includes("cover")) {
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
      if (setTags) {
        const newTagIDs = tagOperation === "merge" ? updatedTags : [];
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

      if (sceneUpdateResult?.data?.sceneUpdate)
        setScene(sceneUpdateResult.data.sceneUpdate);

      queueFingerprintSubmission(stashScene.id, endpoint);
    }

    setSaveState("");
  };

  const classname = cx("row mx-0 mt-2 search-result", {
    "selected-result": isActive,
  });

  const sceneTitle = scene.url ? (
    <a
      href={scene.url}
      target="_blank"
      rel="noopener noreferrer"
      className="scene-link"
    >
      <TruncatedText text={scene?.title} />
    </a>
  ) : (
    <TruncatedText text={scene?.title} />
  );

  const saveEnabled =
    Object.keys(performers ?? []).length ===
      scene.performers.filter((p) => p.gender !== "MALE" || showMales).length &&
    Object.keys(performers ?? []).every((id) => performers?.[id].type) &&
    saveState === "";

  const endpointBase = endpoint.match(/https?:\/\/.*?\//)?.[0];
  const stashBoxURL = endpointBase
    ? `${endpointBase}scenes/${scene.stash_id}`
    : "";

  return (
    // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-noninteractive-element-interactions
    <li
      className={classname}
      key={scene.stash_id}
      onClick={() => !isActive && setActive()}
    >
      <div className="col-lg-6">
        <div className="row">
          <a href={stashBoxURL} target="_blank" rel="noopener noreferrer">
            <img
              src={scene.images[0]}
              alt=""
              className="align-self-center scene-image"
            />
          </a>
          <div className="d-flex flex-column justify-content-center scene-metadata">
            <h4>{sceneTitle}</h4>
            <h5>
              {scene?.studio?.name} • {scene?.date}
            </h5>
            <div>
              {intl.formatMessage(
                { id: "countables.performers" },
                { count: scene?.performers?.length }
              )}
              : {scene?.performers?.map((p) => p.name).join(", ")}
            </div>
            {getDurationStatus(scene, stashScene.file?.duration)}
            {getFingerprintStatus(scene, stashScene)}
          </div>
        </div>
      </div>
      {isActive && (
        <div className="col-lg-6">
          <StudioResult studio={scene.studio} setStudio={setStudio} />
          {scene.performers
            .filter((p) => p.gender !== "MALE" || showMales)
            .map((performer) => (
              <PerformerResult
                performer={performer}
                setPerformer={(data: PerformerOperation) =>
                  setPerformer(data, performer.stash_id)
                }
                key={`${scene.stash_id}${performer.stash_id}`}
                endpoint={endpoint}
              />
            ))}
          <div className="row no-gutters mt-2 align-items-center justify-content-end">
            {error.message && (
              <strong className="mt-1 mr-2 text-danger text-right">
                <abbr title={error.details} className="mr-2">
                  Error:
                </abbr>
                {error.message}
              </strong>
            )}
            {saveState && (
              <strong className="col-4 mt-1 mr-2 text-right">
                {saveState}
              </strong>
            )}
            <Button onClick={handleSave} disabled={!saveEnabled}>
              {saveState ? (
                <LoadingIndicator inline small message="" />
              ) : (
                <FormattedMessage id="actions.save" />
              )}
            </Button>
          </div>
        </div>
      )}
    </li>
  );
};

export default StashSearchResult;
