import React, { useState, useReducer } from "react";
import cx from "classnames";
import { Button } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";

import * as GQL from "src/core/generated-graphql";
import {
  LoadingIndicator,
  SuccessIcon,
  TruncatedText,
} from "src/components/Shared";
import PerformerResult, { PerformerOperation } from "./PerformerResult";
import StudioResult, { StudioOperation } from "./StudioResult";
import { IStashBoxScene } from "./utils";
import { useTagScene } from "./taggerService";

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

  if (!scene.duration && durations.length === 0)
    return <FormattedMessage id="component_tagger.results.duration_unknown" />;

  const minDiff = Math.min(
    Math.abs(scene.duration - stashDuration),
    ...durations
  );
  return (
    <FormattedMessage
      id="component_tagger.results.duration_off"
      values={{ number: Math.floor(minDiff) }}
    />
  );
};

const getFingerprintStatus = (
  scene: IStashBoxScene,
  stashScene: GQL.SlimSceneDataFragment
) => {
  const checksumMatch = scene.fingerprints.some(
    (f) => f.hash === stashScene.checksum || f.hash === stashScene.oshash
  );
  const phashMatch = scene.fingerprints.some(
    (f) => f.hash === stashScene.phash
  );
  if (checksumMatch || phashMatch)
    return (
      <div className="font-weight-bold">
        <SuccessIcon className="mr-2" />
        <FormattedMessage
          id="component_tagger.results.hash_matches"
          values={{
            hash_type: (
              <FormattedMessage
                id={`media_info.${phashMatch ? "phash" : "checksum"}`}
              />
            ),
          }}
        />
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
  setCoverImage: boolean;
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
  setCoverImage,
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

  const tagScene = useTagScene(
    {
      tagOperation,
      setCoverImage,
      setTags,
    },
    setSaveState,
    setError
  );

  async function handleSave() {
    const updatedScene = await tagScene(
      stashScene,
      scene,
      studio,
      performers,
      endpoint
    );

    if (updatedScene) setScene(updatedScene);

    queueFingerprintSubmission(stashScene.id, endpoint);
  }

  const setPerformer = (
    performerData: PerformerOperation,
    performerID: string
  ) => dispatch({ id: performerID, data: performerData });

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
