import React, { useEffect, useRef, useState } from "react";
import { Button, Card } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";

import * as GQL from "src/core/generated-graphql";
import { LoadingIndicator } from "src/components/Shared";
import { stashBoxSceneBatchQuery } from "src/core/StashService";

import { SceneQueue } from "src/models/sceneQueue";
import { ITaggerConfig } from "./constants";
import { selectScenes, IStashBoxScene } from "./utils";
import { TaggerScene } from "./TaggerScene";

interface IFingerprintQueue {
  getQueue: (endpoint: string) => string[];
  queueFingerprintSubmission: (sceneId: string, endpoint: string) => void;
  submitFingerprints: (endpoint: string) => Promise<void> | undefined;
  submittingFingerprints: boolean;
}

interface ITaggerListProps {
  scenes: GQL.SlimSceneDataFragment[];
  queue?: SceneQueue;
  selectedEndpoint: { endpoint: string; index: number };
  config: ITaggerConfig;
  queryScene: (searchVal: string) => Promise<GQL.QueryStashBoxSceneQuery>;
  fingerprintQueue: IFingerprintQueue;
}

// Caches fingerprint lookups between page renders
let fingerprintCache: Record<string, IStashBoxScene[]> = {};

export const TaggerList: React.FC<ITaggerListProps> = ({
  scenes,
  queue,
  selectedEndpoint,
  config,
  queryScene,
  fingerprintQueue,
}) => {
  const intl = useIntl();
  const [fingerprintError, setFingerprintError] = useState("");
  const [loading, setLoading] = useState(false);
  const inputForm = useRef<HTMLFormElement>(null);

  const [searchResults, setSearchResults] = useState<
    Record<string, IStashBoxScene[]>
  >({});
  const [searchErrors, setSearchErrors] = useState<
    Record<string, string | undefined>
  >({});
  const [taggedScenes, setTaggedScenes] = useState<
    Record<string, Partial<GQL.SlimSceneDataFragment>>
  >({});
  const [loadingFingerprints, setLoadingFingerprints] = useState(false);
  const [fingerprints, setFingerprints] = useState<
    Record<string, IStashBoxScene[]>
  >(fingerprintCache);
  const [hideUnmatched, setHideUnmatched] = useState(false);
  const queuedFingerprints = fingerprintQueue.getQueue(
    selectedEndpoint.endpoint
  );

  useEffect(() => {
    inputForm?.current?.reset();
  }, [config.mode, config.blacklist]);

  function clearSceneSearchResult(sceneID: string) {
    // remove sceneID results from the results object
    const { [sceneID]: _removedResult, ...newSearchResults } = searchResults;
    const { [sceneID]: _removedError, ...newSearchErrors } = searchErrors;
    setSearchResults(newSearchResults);
    setSearchErrors(newSearchErrors);
  }

  const doSceneQuery = (sceneID: string, searchVal: string) => {
    clearSceneSearchResult(sceneID);

    queryScene(searchVal)
      .then((queryData) => {
        const s = selectScenes(queryData.queryStashBoxScene);
        setSearchResults({
          ...searchResults,
          [sceneID]: s,
        });
        setSearchErrors({
          ...searchErrors,
          [sceneID]: undefined,
        });
        setLoading(false);
      })
      .catch(() => {
        setLoading(false);
        // Destructure to remove existing result
        const { [sceneID]: unassign, ...results } = searchResults;
        setSearchResults(results);
        setSearchErrors({
          ...searchErrors,
          [sceneID]: "Network Error",
        });
      });

    setLoading(true);
  };

  const handleFingerprintSubmission = () => {
    fingerprintQueue.submitFingerprints(selectedEndpoint.endpoint);
  };

  const handleTaggedScene = (scene: Partial<GQL.SlimSceneDataFragment>) => {
    setTaggedScenes({
      ...taggedScenes,
      [scene.id as string]: scene,
    });
  };

  const handleFingerprintSearch = async () => {
    setLoadingFingerprints(true);
    const newFingerprints = { ...fingerprints };

    const sceneIDs = scenes
      .filter((s) => s.stash_ids.length === 0)
      .map((s) => s.id);

    const results = await stashBoxSceneBatchQuery(
      sceneIDs,
      selectedEndpoint.index
    ).catch(() => {
      setLoadingFingerprints(false);
      setFingerprintError("Network Error");
    });

    if (!results) return;

    // clear search errors
    setSearchErrors({});

    selectScenes(results.data?.queryStashBoxScene).forEach((scene) => {
      scene.fingerprints?.forEach((f) => {
        newFingerprints[f.hash] = newFingerprints[f.hash]
          ? [...newFingerprints[f.hash], scene]
          : [scene];
      });
    });

    // Null any ids that are still undefined since it means they weren't found
    sceneIDs.forEach((id) => {
      newFingerprints[id] = newFingerprints[id] ?? null;
    });

    setFingerprints(newFingerprints);
    fingerprintCache = newFingerprints;
    setLoadingFingerprints(false);
    setFingerprintError("");
  };

  const canFingerprintSearch = () =>
    scenes.some(
      (s) => s.stash_ids.length === 0 && fingerprints[s.id] === undefined
    );

  const getFingerprintCount = () => {
    return scenes.filter(
      (s) =>
        s.stash_ids.length === 0 &&
        ((s.checksum && fingerprints[s.checksum]) ||
          (s.oshash && fingerprints[s.oshash]) ||
          (s.phash && fingerprints[s.phash]))
    ).length;
  };

  const getFingerprintCountMessage = () => {
    const count = getFingerprintCount();
    return intl.formatMessage(
      { id: "component_tagger.results.fp_found" },
      { fpCount: count }
    );
  };

  const toggleHideUnmatchedScenes = () => {
    setHideUnmatched(!hideUnmatched);
  };

  function generateSceneLink(scene: GQL.SlimSceneDataFragment, index: number) {
    return queue
      ? queue.makeLink(scene.id, { sceneIndex: index })
      : `/scenes/${scene.id}`;
  }

  const renderScenes = () =>
    scenes.map((scene, index) => {
      const sceneLink = generateSceneLink(scene, index);
      const searchResult = {
        results: searchResults[scene.id],
        error: searchErrors[scene.id],
      };

      return (
        <TaggerScene
          key={scene.id}
          config={config}
          endpoint={selectedEndpoint.endpoint}
          queueFingerprintSubmission={
            fingerprintQueue.queueFingerprintSubmission
          }
          scene={scene}
          url={sceneLink}
          hideUnmatched={hideUnmatched}
          loading={loading}
          taggedScene={taggedScenes[scene.id]}
          doSceneQuery={(queryString) => doSceneQuery(scene.id, queryString)}
          tagScene={handleTaggedScene}
          searchResult={searchResult}
        />
      );
    });

  return (
    <Card className="tagger-table">
      <div className="tagger-table-header d-flex flex-nowrap align-items-center">
        <b className="ml-auto mr-2 text-danger">{fingerprintError}</b>
        <div className="mr-2">
          {(getFingerprintCount() > 0 || hideUnmatched) && (
            <Button onClick={toggleHideUnmatchedScenes}>
              <FormattedMessage
                id="component_tagger.verb_toggle_unmatched"
                values={{
                  toggle: (
                    <FormattedMessage
                      id={`actions.${hideUnmatched ? "hide" : "show"}`}
                    />
                  ),
                }}
              />
            </Button>
          )}
        </div>
        <div className="mr-2">
          {queuedFingerprints.length > 0 && (
            <Button
              onClick={handleFingerprintSubmission}
              disabled={fingerprintQueue.submittingFingerprints}
            >
              {fingerprintQueue.submittingFingerprints ? (
                <LoadingIndicator message="" inline small />
              ) : (
                <span>
                  <FormattedMessage
                    id="component_tagger.verb_submit_fp"
                    values={{ fpCount: queuedFingerprints.length }}
                  />
                </span>
              )}
            </Button>
          )}
        </div>
        <Button
          onClick={handleFingerprintSearch}
          disabled={!canFingerprintSearch() && !loadingFingerprints}
        >
          {canFingerprintSearch() && (
            <span>
              {intl.formatMessage({ id: "component_tagger.verb_match_fp" })}
            </span>
          )}
          {!canFingerprintSearch() && getFingerprintCountMessage()}
          {loadingFingerprints && <LoadingIndicator message="" inline small />}
        </Button>
      </div>
      <form ref={inputForm}>{renderScenes()}</form>
    </Card>
  );
};
