import React, { useState } from "react";
import { Button } from "react-bootstrap";
import { FormattedMessage } from "react-intl";
import { HashLink } from "react-router-hash-link";
import { useLocalForage } from "src/hooks";

import * as GQL from "src/core/generated-graphql";
import { LoadingIndicator } from "src/components/Shared";
import { stashBoxSceneQuery, useConfiguration } from "src/core/StashService";
import { Manual } from "src/components/Help/Manual";

import { SceneQueue } from "src/models/sceneQueue";
import Config from "./Config";
import { LOCAL_FORAGE_KEY, ITaggerConfig, initialConfig } from "./constants";
import { TaggerList } from "./TaggerList";

interface ITaggerProps {
  scenes: GQL.SlimSceneDataFragment[];
  queue?: SceneQueue;
}

export const Tagger: React.FC<ITaggerProps> = ({ scenes, queue }) => {
  const stashConfig = useConfiguration();
  const [{ data: config }, setConfig] = useLocalForage<ITaggerConfig>(
    LOCAL_FORAGE_KEY,
    initialConfig
  );
  const [showConfig, setShowConfig] = useState(false);
  const [showManual, setShowManual] = useState(false);

  const clearSubmissionQueue = (endpoint: string) => {
    if (!config) return;

    setConfig({
      ...config,
      fingerprintQueue: {
        ...config.fingerprintQueue,
        [endpoint]: [],
      },
    });
  };

  const [
    submitFingerprints,
    { loading: submittingFingerprints },
  ] = GQL.useSubmitStashBoxFingerprintsMutation();

  const handleFingerprintSubmission = (endpoint: string) => {
    if (!config) return;

    return submitFingerprints({
      variables: {
        input: {
          stash_box_index: getEndpointIndex(endpoint),
          scene_ids: config?.fingerprintQueue[endpoint],
        },
      },
    }).then(() => {
      clearSubmissionQueue(endpoint);
    });
  };

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

  function getEndpointIndex(endpoint: string) {
    return (
      stashConfig.data?.configuration.general.stashBoxes.findIndex(
        (s) => s.endpoint === endpoint
      ) ?? -1
    );
  }

  async function doBoxSearch(searchVal: string) {
    return (await stashBoxSceneQuery(searchVal, selectedEndpointIndex)).data;
  }

  const queueFingerprintSubmission = (sceneId: string, endpoint: string) => {
    if (!config) return;
    setConfig({
      ...config,
      fingerprintQueue: {
        ...config.fingerprintQueue,
        [endpoint]: [...(config.fingerprintQueue[endpoint] ?? []), sceneId],
      },
    });
  };

  const getQueue = (endpoint: string) => {
    if (!config) return [];
    return config.fingerprintQueue[endpoint] ?? [];
  };

  const fingerprintQueue = {
    queueFingerprintSubmission,
    getQueue,
    submitFingerprints: handleFingerprintSubmission,
    submittingFingerprints,
  };

  return (
    <>
      <Manual
        show={showManual}
        onClose={() => setShowManual(false)}
        defaultActiveTab="Tagger.md"
      />
      <div className="tagger-container mx-md-auto">
        {selectedEndpointIndex !== -1 && selectedEndpoint ? (
          <>
            <div className="row mb-2 no-gutters">
              <Button onClick={() => setShowConfig(!showConfig)} variant="link">
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
              queryScene={doBoxSearch}
              fingerprintQueue={fingerprintQueue}
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
