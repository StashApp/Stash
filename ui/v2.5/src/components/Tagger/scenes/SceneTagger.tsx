import React, { useEffect, useRef, useState } from "react";
import { Button, Card, Form, InputGroup, ProgressBar } from "react-bootstrap";
import { Link } from "react-router-dom";
import { FormattedMessage, useIntl } from "react-intl";
import { HashLink } from "react-router-hash-link";
import { uniqBy } from "lodash";
import distance from "hamming-distance";

import { ScenePreview } from "src/components/Scenes/SceneCard";
import { useLocalForage } from "src/hooks";

import * as GQL from "src/core/generated-graphql";
import { LoadingIndicator, TruncatedText } from "src/components/Shared";
import {
  stashBoxSceneQuery,
  useConfiguration,
  useJobsSubscribe,
  mutateStashBoxBatchSceneTag,
  stashBoxSceneBatchQuery,
} from "src/core/StashService";
import { Manual } from "src/components/Help/Manual";

import { SceneQueue } from "src/models/sceneQueue";
import StashSearchResult from "./StashSearchResult";
import Config from "./Config";
import BatchModal from "./BatchModal";
import {
  LOCAL_FORAGE_KEY,
  ITaggerConfig,
  ParseMode,
  initialConfig,
} from "../constants";
import {
  parsePath,
  selectScenes,
  IStashBoxScene,
  sortScenesByDuration,
} from "../utils";

type JobFragment = Pick<
  GQL.Job,
  "id" | "status" | "subTasks" | "description" | "progress"
>;

const months = [
  "jan",
  "feb",
  "mar",
  "apr",
  "may",
  "jun",
  "jul",
  "aug",
  "sep",
  "oct",
  "nov",
  "dec",
];

const ddmmyyRegex = /\.(\d\d)\.(\d\d)\.(\d\d)\./;
const yyyymmddRegex = /(\d{4})[-.](\d{2})[-.](\d{2})/;
const mmddyyRegex = /(\d{2})[-.](\d{2})[-.](\d{4})/;
const ddMMyyRegex = new RegExp(
  `(\\d{1,2}).(${months.join("|")})\\.?.(\\d{4})`,
  "i"
);
const MMddyyRegex = new RegExp(
  `(${months.join("|")})\\.?.(\\d{1,2}),?.(\\d{4})`,
  "i"
);
const parseDate = (input: string): string => {
  let output = input;
  const ddmmyy = output.match(ddmmyyRegex);
  if (ddmmyy) {
    output = output.replace(
      ddmmyy[0],
      ` 20${ddmmyy[1]}-${ddmmyy[2]}-${ddmmyy[3]} `
    );
  }
  const mmddyy = output.match(mmddyyRegex);
  if (mmddyy) {
    output = output.replace(
      mmddyy[0],
      ` ${mmddyy[1]}-${mmddyy[2]}-${mmddyy[3]} `
    );
  }
  const ddMMyy = output.match(ddMMyyRegex);
  if (ddMMyy) {
    const month = (months.indexOf(ddMMyy[2].toLowerCase()) + 1)
      .toString()
      .padStart(2, "0");
    output = output.replace(
      ddMMyy[0],
      ` ${ddMMyy[3]}-${month}-${ddMMyy[1].padStart(2, "0")} `
    );
  }
  const MMddyy = output.match(MMddyyRegex);
  if (MMddyy) {
    const month = (months.indexOf(MMddyy[1].toLowerCase()) + 1)
      .toString()
      .padStart(2, "0");
    output = output.replace(
      MMddyy[0],
      ` ${MMddyy[3]}-${month}-${MMddyy[2].padStart(2, "0")} `
    );
  }

  const yyyymmdd = output.search(yyyymmddRegex);
  if (yyyymmdd !== -1)
    return (
      output.slice(0, yyyymmdd).replace(/-/g, " ") +
      output.slice(yyyymmdd, yyyymmdd + 10).replace(/\./g, "-") +
      output.slice(yyyymmdd + 10).replace(/-/g, " ")
    );
  return output.replace(/-/g, " ");
};

function prepareQueryString(
  scene: Partial<GQL.SlimSceneDataFragment>,
  paths: string[],
  filename: string,
  mode: ParseMode,
  blacklist: string[]
) {
  if ((mode === "auto" && scene.date && scene.studio) || mode === "metadata") {
    let str = [
      scene.date,
      scene.studio?.name ?? "",
      (scene?.performers ?? []).map((p) => p.name).join(" "),
      scene?.title ? scene.title.replace(/[^a-zA-Z0-9 ]+/g, "") : "",
    ]
      .filter((s) => s !== "")
      .join(" ");
    blacklist.forEach((b) => {
      str = str.replace(new RegExp(b, "gi"), " ");
    });
    return str;
  }
  let s = "";

  if (mode === "auto" || mode === "filename") {
    s = filename;
  } else if (mode === "path") {
    s = [...paths, filename].join(" ");
  } else {
    s = paths[paths.length - 1];
  }
  blacklist.forEach((b) => {
    s = s.replace(new RegExp(b, "gi"), " ");
  });
  s = parseDate(s);
  return s.replace(/\./g, " ");
}

interface ITaggerListProps {
  scenes: GQL.SlimSceneDataFragment[];
  queue?: SceneQueue;
  selectedEndpoint: { endpoint: string; index: number };
  isIdle: boolean;
  config: ITaggerConfig;
  queueFingerprintSubmission: (sceneId: string, endpoint: string) => void;
  clearSubmissionQueue: (endpoint: string) => void;
  onBatchUpdate: (ids: string[] | undefined, refresh: boolean) => void;
}

// Caches fingerprint lookups between page renders
let fingerprintCache: Record<string, IStashBoxScene[]> = {};

const TaggerList: React.FC<ITaggerListProps> = ({
  scenes,
  queue,
  selectedEndpoint,
  isIdle,
  config,
  queueFingerprintSubmission,
  clearSubmissionQueue,
  onBatchUpdate,
}) => {
  const intl = useIntl();
  const [fingerprintError, setFingerprintError] = useState("");
  const [loading, setLoading] = useState(false);
  const queryString = useRef<Record<string, string>>({});
  const inputForm = useRef<HTMLFormElement>(null);

  const [searchResults, setSearchResults] = useState<
    Record<string, IStashBoxScene[]>
  >({});
  const [searchErrors, setSearchErrors] = useState<
    Record<string, string | undefined>
  >({});
  const [selectedResult, setSelectedResult] = useState<
    Record<string, number>
  >();
  const [selectedFingerprintResult, setSelectedFingerprintResult] = useState<
    Record<string, number>
  >();
  const [taggedScenes, setTaggedScenes] = useState<
    Record<string, Partial<GQL.SlimSceneDataFragment>>
  >({});
  const [loadingFingerprints, setLoadingFingerprints] = useState(false);
  const [fingerprints, setFingerprints] = useState<
    Record<string, IStashBoxScene[]>
  >(fingerprintCache);
  const [hideUnmatched, setHideUnmatched] = useState(false);
  const fingerprintQueue =
    config.fingerprintQueue[selectedEndpoint.endpoint] ?? [];

  const [refresh, setRefresh] = useState(false);
  const { data: allScenes } = GQL.useFindScenesQuery({
    variables: {
      scene_filter: {
        stash_id: {
          value: "",
          modifier: refresh
            ? GQL.CriterionModifier.NotNull
            : GQL.CriterionModifier.IsNull,
        },
      },
      filter: {
        per_page: 0,
      },
    },
  });
  const [showBatchUpdate, setShowBatchUpdate] = useState(false);
  const [queryAll, setQueryAll] = useState(false);

  useEffect(() => {
    inputForm?.current?.reset();
  }, [config.mode, config.blacklist]);

  const doBoxSearch = (sceneID: string, searchVal: string) => {
    stashBoxSceneQuery(searchVal, selectedEndpoint.index)
      .then((queryData) => {
        const s = selectScenes(queryData.data?.queryStashBoxScene);
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

  const [
    submitFingerPrints,
    { loading: submittingFingerprints },
  ] = GQL.useSubmitStashBoxFingerprintsMutation({
    onCompleted: (result) => {
      setFingerprintError("");
      if (result.submitStashBoxFingerprints)
        clearSubmissionQueue(selectedEndpoint.endpoint);
    },
    onError: () => {
      setFingerprintError("Network Error");
    },
  });

  const handleFingerprintSubmission = () => {
    submitFingerPrints({
      variables: {
        input: {
          stash_box_index: selectedEndpoint.index,
          scene_ids: fingerprintQueue,
        },
      },
    });
  };

  const handleBatchUpdate = () => {
    onBatchUpdate(!queryAll ? scenes.map((p) => p.id) : undefined, refresh);
    setShowBatchUpdate(false);
  };

  const handleTaggedScene = (scene: Partial<GQL.SlimSceneDataFragment>) => {
    setTaggedScenes({
      ...taggedScenes,
      [scene.id as string]: scene,
    });
  };

  const handleFingerprintSearch = async () => {
    setLoadingFingerprints(true);

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

    const sceneResults = selectScenes(results.data.queryStashBoxScene);
    const hashes = sceneResults.reduce(
      (dict: Record<string, IStashBoxScene>, scene: IStashBoxScene) => ({
        ...dict,
        ...Object.fromEntries(
          scene.fingerprints
            .filter((f) => f.algorithm !== "PHASH")
            .map((f) => [f.hash, scene])
        ),
      }),
      {}
    );
    const phashes = sceneResults.reduce(
      (dict: Record<string, IStashBoxScene>, scene: IStashBoxScene) => ({
        ...dict,
        ...Object.fromEntries(
          scene.fingerprints
            .filter((f) => f.algorithm === "PHASH")
            .map((f) => [f.hash, scene])
        ),
      }),
      {}
    );

    const sceneHashes = Object.fromEntries(
      scenes.map((s) => [
        s.id,
        uniqBy(
          [
            ...(s.oshash && hashes[s.oshash] ? [hashes[s.oshash]] : []),
            ...(s.checksum && hashes[s.checksum] ? [hashes[s.checksum]] : []),
            ...(s.phash
              ? Object.keys(phashes)
                  .filter((fp) => distance(fp, s.phash) <= 8)
                  .map((fp) => phashes[fp])
              : []),
          ],
          (fpScene) => fpScene.stash_id
        ),
      ])
    );

    const newFingerprints = {
      ...fingerprints,
      ...sceneHashes,
    };
    setFingerprints(newFingerprints);
    fingerprintCache = newFingerprints;
    setLoadingFingerprints(false);
    setFingerprintError("");
  };

  const canFingerprintSearch = () =>
    scenes.some(
      (s) => s.stash_ids.length === 0 && fingerprints[s.id] === undefined
    );

  const getFingerprintCount = () =>
    scenes.filter(
      (s) =>
        s.stash_ids.length === 0 &&
        fingerprints[s.id] &&
        fingerprints[s.id].length > 0
    ).length;

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
      const { paths, file, ext } = parsePath(scene.path);
      const originalDir = scene.path.slice(
        0,
        scene.path.length - file.length - ext.length
      );
      const defaultQueryString = prepareQueryString(
        scene,
        paths,
        file,
        config.mode,
        config.blacklist
      );

      // Get all scenes matching one of the fingerprints, and return array of unique scenes
      const fingerprintMatches = fingerprints[scene.id] ?? [];

      const isTagged = taggedScenes[scene.id];
      const hasStashIDs = scene.stash_ids.length > 0;
      const width = scene.file.width ? scene.file.width : 0;
      const height = scene.file.height ? scene.file.height : 0;
      const isPortrait = height > width;

      let mainContent;
      if (!isTagged && hasStashIDs) {
        mainContent = (
          <div className="text-right">
            <h5 className="text-bold">
              <FormattedMessage id="component_tagger.results.match_failed_already_tagged" />
            </h5>
          </div>
        );
      } else if (!isTagged && !hasStashIDs) {
        mainContent = (
          <InputGroup>
            <InputGroup.Prepend>
              <InputGroup.Text>
                <FormattedMessage id="component_tagger.noun_query" />
              </InputGroup.Text>
            </InputGroup.Prepend>
            <Form.Control
              className="text-input"
              defaultValue={queryString.current[scene.id] || defaultQueryString}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                queryString.current[scene.id] = e.currentTarget.value;
              }}
              onKeyPress={(e: React.KeyboardEvent<HTMLInputElement>) =>
                e.key === "Enter" &&
                doBoxSearch(
                  scene.id,
                  queryString.current[scene.id] || defaultQueryString
                )
              }
            />
            <InputGroup.Append>
              <Button
                disabled={loading}
                onClick={() =>
                  doBoxSearch(
                    scene.id,
                    queryString.current[scene.id] || defaultQueryString
                  )
                }
              >
                <FormattedMessage id="actions.search" />
              </Button>
            </InputGroup.Append>
          </InputGroup>
        );
      } else if (isTagged) {
        mainContent = (
          <div className="d-flex flex-column text-right">
            <h5>
              <FormattedMessage id="component_tagger.results.match_success" />
            </h5>
            <h6>
              <Link className="bold" to={sceneLink}>
                {taggedScenes[scene.id].title}
              </Link>
            </h6>
          </div>
        );
      }

      let subContent;
      if (scene.stash_ids.length > 0) {
        const stashLinks = scene.stash_ids.map((stashID) => {
          const base = stashID.endpoint.match(/https?:\/\/.*?\//)?.[0];
          const link = base ? (
            <a
              className="small d-block"
              href={`${base}scenes/${stashID.stash_id}`}
              target="_blank"
              rel="noopener noreferrer"
            >
              {stashID.stash_id}
            </a>
          ) : (
            <div className="small">{stashID.stash_id}</div>
          );

          return link;
        });
        subContent = <>{stashLinks}</>;
      } else if (searchErrors[scene.id]) {
        subContent = (
          <div className="text-danger font-weight-bold">
            {searchErrors[scene.id]}
          </div>
        );
      } else if (searchResults[scene.id]?.length === 0) {
        subContent = (
          <div className="text-danger font-weight-bold">
            <FormattedMessage id="component_tagger.results.match_failed_no_result" />
          </div>
        );
      }

      let searchResult;
      if (fingerprintMatches.length > 0 && !isTagged && !hasStashIDs) {
        searchResult = sortScenesByDuration(
          fingerprintMatches,
          scene.file.duration ?? 0
        ).map((match, i) => (
          <StashSearchResult
            showMales={config.showMales}
            stashScene={scene}
            isActive={(selectedFingerprintResult?.[scene.id] ?? 0) === i}
            setActive={() =>
              setSelectedFingerprintResult({
                ...selectedFingerprintResult,
                [scene.id]: i,
              })
            }
            setScene={handleTaggedScene}
            scene={match}
            excludedFields={config.excludedSceneFields}
            setTags={config.setTags}
            tagOperation={config.tagOperation}
            endpoint={selectedEndpoint.endpoint}
            queueFingerprintSubmission={queueFingerprintSubmission}
            key={match.stash_id}
          />
        ));
      } else if (
        searchResults[scene.id]?.length > 0 &&
        !isTagged &&
        fingerprintMatches.length === 0
      ) {
        searchResult = (
          <ul className="pl-0 mt-3 mb-0">
            {sortScenesByDuration(
              searchResults[scene.id],
              scene.file.duration ?? undefined
            ).map(
              (sceneResult, i) =>
                sceneResult && (
                  <StashSearchResult
                    key={sceneResult.stash_id}
                    showMales={config.showMales}
                    stashScene={scene}
                    scene={sceneResult}
                    isActive={(selectedResult?.[scene.id] ?? 0) === i}
                    setActive={() =>
                      setSelectedResult({
                        ...selectedResult,
                        [scene.id]: i,
                      })
                    }
                    excludedFields={config.excludedSceneFields}
                    tagOperation={config.tagOperation}
                    setTags={config.setTags}
                    setScene={handleTaggedScene}
                    endpoint={selectedEndpoint.endpoint}
                    queueFingerprintSubmission={queueFingerprintSubmission}
                  />
                )
            )}
          </ul>
        );
      }

      return hideUnmatched && fingerprintMatches.length === 0 ? null : (
        <div key={scene.id} className="mt-3 search-item">
          <div className="row">
            <div className="col col-lg-6 overflow-hidden align-items-center d-flex flex-column flex-sm-row">
              <div className="scene-card mr-3">
                <Link to={sceneLink}>
                  <ScenePreview
                    image={scene.paths.screenshot ?? undefined}
                    video={scene.paths.preview ?? undefined}
                    isPortrait={isPortrait}
                    soundActive={false}
                  />
                </Link>
              </div>
              <Link to={sceneLink} className="scene-link overflow-hidden">
                <TruncatedText
                  text={`${originalDir}\u200B${file}${ext}`}
                  lineCount={2}
                />
              </Link>
            </div>
            <div className="col-md-6 my-1 align-self-center">
              {mainContent}
              <div className="sub-content text-right">{subContent}</div>
            </div>
          </div>
          {searchResult}
        </div>
      );
    });

  const batchSceneCount = queryAll
    ? allScenes?.findScenes.count ?? 0
    : scenes.filter((p) =>
        refresh ? p.stash_ids.length > 0 : p.stash_ids.length === 0
      ).length;

  return (
    <Card className="tagger-table">
      <BatchModal
        show={showBatchUpdate}
        hide={() => setShowBatchUpdate(false)}
        handleBatchUpdate={handleBatchUpdate}
        isIdle={isIdle}
        sceneCount={batchSceneCount}
        setQueryAll={setQueryAll}
        setRefresh={setRefresh}
      />
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
          {fingerprintQueue.length > 0 && (
            <Button
              onClick={handleFingerprintSubmission}
              disabled={submittingFingerprints}
            >
              {submittingFingerprints ? (
                <LoadingIndicator message="" inline small />
              ) : (
                <span>
                  <FormattedMessage
                    id="component_tagger.verb_submit_fp"
                    values={{ fpCount: fingerprintQueue.length }}
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
        <Button className="ml-3" onClick={() => setShowBatchUpdate(true)}>
          Batch Update Scenes
        </Button>
      </div>
      <form ref={inputForm}>{renderScenes()}</form>
    </Card>
  );
};

interface ITaggerProps {
  scenes: GQL.SlimSceneDataFragment[];
  queue?: SceneQueue;
}

export const Tagger: React.FC<ITaggerProps> = ({ scenes, queue }) => {
  const jobsSubscribe = useJobsSubscribe();
  const stashConfig = useConfiguration();
  const [{ data: config }, setConfig] = useLocalForage<ITaggerConfig>(
    LOCAL_FORAGE_KEY,
    initialConfig
  );
  const [showConfig, setShowConfig] = useState(false);
  const [showManual, setShowManual] = useState(false);

  const [batchJob, setBatchJob] = useState<JobFragment | undefined>();

  // monitor batch operation
  useEffect(() => {
    if (!jobsSubscribe.data) {
      return;
    }

    const event = jobsSubscribe.data.jobsSubscribe;
    if (event.job.description !== "Batch stash-box scene tag...") {
      return;
    }

    if (event.type !== GQL.JobStatusUpdateType.Remove) {
      setBatchJob(event.job);
    } else {
      setBatchJob(undefined);
    }
  }, [jobsSubscribe]);

  if (!config) return <LoadingIndicator />;

  const savedEndpointIndex =
    stashConfig.data?.configuration.general.stashBoxes.findIndex(
      (s) => s.endpoint === config.selectedEndpoint
    ) ?? -1;
  const selectedEndpointIndex =
    savedEndpointIndex === -1 &&
    stashConfig.data?.configuration.general.stashBoxes.length
      ? 0
      : savedEndpointIndex;
  const selectedEndpoint =
    stashConfig.data?.configuration.general.stashBoxes[selectedEndpointIndex];

  const queueFingerprintSubmission = (sceneId: string, endpoint: string) => {
    setConfig({
      ...config,
      fingerprintQueue: {
        ...config.fingerprintQueue,
        [endpoint]: [...(config.fingerprintQueue[endpoint] ?? []), sceneId],
      },
    });
  };

  const clearSubmissionQueue = (endpoint: string) => {
    setConfig({
      ...config,
      fingerprintQueue: {
        ...config.fingerprintQueue,
        [endpoint]: [],
      },
    });
  };

  async function batchUpdate(ids: string[] | undefined, refresh: boolean) {
    if (config && selectedEndpoint) {
      await mutateStashBoxBatchSceneTag({
        scene_ids: ids,
        endpoint: selectedEndpointIndex,
        refresh,
        set_organized: config.setOrganized ?? false,
        tag_strategy: !config.setTags
          ? GQL.TagStrategy.Ignore
          : config.tagOperation === "overwrite"
          ? GQL.TagStrategy.Overwrite
          : GQL.TagStrategy.Merge,
        create_tags: config.createTags ?? false,
        exclude_fields: config.excludedSceneFields ?? [],
        tag_male_performers: config.showMales ?? false,
      });
    }
  }

  function renderStatus() {
    if (batchJob) {
      const progress =
        batchJob.progress !== undefined && batchJob.progress !== null
          ? batchJob.progress * 100
          : undefined;
      return (
        <Form.Group className="py-4 px-2">
          <h5>Status: {batchJob.description}</h5>
          {progress !== undefined && (
            <ProgressBar
              animated
              now={progress}
              label={`${progress.toFixed(0)}%`}
            />
          )}
          {(batchJob.subTasks ?? []).length > 0 && (
            <div>{batchJob.subTasks?.join(", ")}</div>
          )}
        </Form.Group>
      );
    }
  }

  return (
    <>
      <Manual
        show={showManual}
        onClose={() => setShowManual(false)}
        defaultActiveTab="Tagger.md"
      />
      <div className="tagger-container mx-md-auto">
        {renderStatus()}
        {selectedEndpointIndex !== -1 && selectedEndpoint ? (
          <>
            <div className="row mb-2 no-gutters">
              <Button
                onClick={() => setShowConfig(!showConfig)}
                variant="primary"
                className="ml-2"
              >
                <FormattedMessage
                  id="component_tagger.verb_toggle_config"
                  values={{
                    toggle: (
                      <FormattedMessage
                        id={`actions.${showConfig ? "hide" : "show"}`}
                      />
                    ),
                    configuration: <FormattedMessage id="configuration" />,
                  }}
                />
              </Button>
              <Button
                className="ml-auto"
                onClick={() => setShowManual(true)}
                title="Help"
                variant="link"
              >
                <FormattedMessage id="help" />
              </Button>
            </div>

            <Config config={config} setConfig={setConfig} show={showConfig} />
            <TaggerList
              scenes={scenes}
              queue={queue}
              config={config}
              selectedEndpoint={{
                endpoint: selectedEndpoint.endpoint,
                index: selectedEndpointIndex,
              }}
              isIdle={!batchJob}
              queueFingerprintSubmission={queueFingerprintSubmission}
              clearSubmissionQueue={clearSubmissionQueue}
              onBatchUpdate={batchUpdate}
            />
          </>
        ) : (
          <div className="my-4">
            <h3 className="text-center mt-4">
              To use the scene tagger a stash-box instance needs to be
              configured.
            </h3>
            <h5 className="text-center">
              Please see{" "}
              <HashLink
                to="/settings?tab=configuration#stashbox"
                scroll={(el) =>
                  el.scrollIntoView({ behavior: "smooth", block: "center" })
                }
              >
                Settings.
              </HashLink>
            </h5>
          </div>
        )}
      </div>
    </>
  );
};
