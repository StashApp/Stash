import React, { useEffect, useState } from "react";
import { Button, ButtonGroup } from "react-bootstrap";
import { FormattedMessage } from "react-intl";
import cx from "classnames";

import { SuccessIcon, PerformerSelect } from "src/components/Shared";
import * as GQL from "src/core/generated-graphql";
import { ValidTypes } from "src/components/Shared/Select";
import { IStashBoxPerformer, filterPerformer } from "./utils";

import PerformerModal from "./PerformerModal";

export type PerformerOperation =
  | { type: "create"; data: IStashBoxPerformer }
  | { type: "update"; data: GQL.SlimPerformerDataFragment }
  | { type: "existing"; data: GQL.PerformerDataFragment }
  | { type: "skip" };

export interface IPerformerOperations {
  [x: string]: PerformerOperation;
}

interface IPerformerResultProps {
  performer: IStashBoxPerformer;
  setPerformer: (data: PerformerOperation) => void;
  endpoint: string;
}

const PerformerResult: React.FC<IPerformerResultProps> = ({
  performer,
  setPerformer,
  endpoint,
}) => {
  const [selectedPerformer, setSelectedPerformer] = useState<string | null>();
  const [selectedSource, setSelectedSource] = useState<
    "create" | "existing" | "skip" | undefined
  >();
  const [modalVisible, showModal] = useState(false);
  const { data: performerData } = GQL.useFindPerformerQuery({
    variables: { id: performer.id ?? "" },
    skip: !performer.id,
  });
  const { data: stashData, loading: stashLoading } = GQL.useFindPerformersQuery(
    {
      variables: {
        performer_filter: {
          stash_id: {
            value: performer.stash_id,
            modifier: GQL.CriterionModifier.Equals,
          },
        },
      },
    }
  );

  useEffect(() => {
    if (stashData?.findPerformers.performers.length)
      setPerformer({
        type: "existing",
        data: stashData.findPerformers.performers[0],
      });
    else if (performerData?.findPerformer) {
      setSelectedPerformer(performerData.findPerformer.id);
      setSelectedSource("existing");
      setPerformer({
        type: "update",
        data: performerData.findPerformer,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stashData, performerData]);

  const handlePerformerSelect = (performers: ValidTypes[]) => {
    if (performers.length) {
      setSelectedSource("existing");
      setSelectedPerformer(performers[0].id);
      setPerformer({
        type: "update",
        data: performers[0] as GQL.SlimPerformerDataFragment,
      });
    } else {
      setSelectedSource(undefined);
      setSelectedPerformer(null);
    }
  };

  const handlePerformerCreate = (
    imageIndex: number,
    excludedFields: string[]
  ) => {
    const selectedImage = performer.images[imageIndex];
    const images = selectedImage ? [selectedImage] : [];

    setSelectedSource("create");
    setPerformer({
      type: "create",
      data: {
        ...filterPerformer(performer, excludedFields),
        name: performer.name,
        stash_id: performer.stash_id,
        images,
      },
    });
    showModal(false);
  };

  const handlePerformerSkip = () => {
    setSelectedSource("skip");
    setPerformer({
      type: "skip",
    });
  };

  if (stashLoading) return <div>Loading performer</div>;

  if (stashData?.findPerformers.performers?.[0]?.id) {
    return (
      <div className="row no-gutters my-2">
        <div className="entity-name">
          <FormattedMessage id="countables.performers" values={{ count: 1 }} />:
          <b className="ml-2">{performer.name}</b>
        </div>
        <span className="ml-auto">
          <SuccessIcon />
          <FormattedMessage id="component_tagger.verb_matched" />:
        </span>
        <b className="col-3 text-right">
          {stashData.findPerformers.performers[0].name}
        </b>
      </div>
    );
  }
  return (
    <div className="row no-gutters align-items-center mt-2">
      <PerformerModal
        closeModal={() => showModal(false)}
        modalVisible={modalVisible}
        performer={performer}
        handlePerformerCreate={handlePerformerCreate}
        icon="star"
        header="Create Performer"
        create
        endpoint={endpoint}
      />
      <div className="entity-name">
        <FormattedMessage id="countables.performers" values={{ count: 1 }} />:
        <b className="ml-2">{performer.name}</b>
      </div>
      <ButtonGroup>
        <Button
          variant={selectedSource === "create" ? "primary" : "secondary"}
          onClick={() => showModal(true)}
        >
          <FormattedMessage id="actions.create" />
        </Button>
        <Button
          variant={selectedSource === "skip" ? "primary" : "secondary"}
          onClick={() => handlePerformerSkip()}
        >
          <FormattedMessage id="actions.skip" />
        </Button>
        <PerformerSelect
          ids={selectedPerformer ? [selectedPerformer] : []}
          onSelect={handlePerformerSelect}
          className={cx("performer-select", {
            "performer-select-active": selectedSource === "existing",
          })}
          isClearable={false}
        />
      </ButtonGroup>
    </div>
  );
};

export default PerformerResult;
