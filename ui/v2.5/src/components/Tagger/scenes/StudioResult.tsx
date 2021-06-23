import React, { useEffect, useState, Dispatch, SetStateAction } from "react";
import { Button, ButtonGroup } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import cx from "classnames";

import { SuccessIcon, Modal, StudioSelect } from "src/components/Shared";
import * as GQL from "src/core/generated-graphql";
import { ValidTypes } from "src/components/Shared/Select";
import { IStashBoxStudio } from "../utils";

export type StudioOperation =
  | { type: "create"; data: IStashBoxStudio }
  | { type: "update"; data: GQL.SlimStudioDataFragment }
  | { type: "existing"; data: GQL.StudioDataFragment }
  | { type: "skip" };

interface IStudioResultProps {
  studio: IStashBoxStudio | null;
  setStudio: Dispatch<SetStateAction<StudioOperation | undefined>>;
}

const StudioResult: React.FC<IStudioResultProps> = ({ studio, setStudio }) => {
  const intl = useIntl();
  const [selectedStudio, setSelectedStudio] = useState<string | null>();
  const [modalVisible, showModal] = useState(false);
  const [selectedSource, setSelectedSource] = useState<
    "create" | "existing" | "skip" | undefined
  >();
  const { data: studioData } = GQL.useFindStudioQuery({
    variables: { id: studio?.id ?? "" },
    skip: !studio?.id,
  });
  const {
    data: stashIDData,
    loading: loadingStashID,
  } = GQL.useFindStudiosQuery({
    variables: {
      studio_filter: {
        stash_id: {
          value: studio?.stash_id ?? "no-stashid",
          modifier: GQL.CriterionModifier.Equals,
        },
      },
    },
  });

  useEffect(() => {
    if (stashIDData?.findStudios.studios?.[0])
      setStudio({
        type: "existing",
        data: stashIDData.findStudios.studios[0],
      });
    else if (studioData?.findStudio) {
      setSelectedSource("existing");
      setSelectedStudio(studioData.findStudio.id);
      setStudio({
        type: "update",
        data: studioData.findStudio,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stashIDData, studioData]);

  const handleStudioSelect = (newStudio: ValidTypes[]) => {
    if (newStudio.length) {
      setSelectedSource("existing");
      setSelectedStudio(newStudio[0].id);
      setStudio({
        type: "update",
        data: newStudio[0] as GQL.SlimStudioDataFragment,
      });
    } else {
      setSelectedSource(undefined);
      setSelectedStudio(null);
    }
  };

  const handleStudioCreate = () => {
    if (!studio) return;
    setSelectedSource("create");
    setStudio({
      type: "create",
      data: studio,
    });
    showModal(false);
  };

  const handleStudioSkip = () => {
    setSelectedSource("skip");
    setStudio({ type: "skip" });
  };

  if (loadingStashID) return <div>Loading studio</div>;

  if (stashIDData?.findStudios.studios.length) {
    return (
      <div className="row no-gutters my-2">
        <div className="entity-name">
          <FormattedMessage
            id="countables.studios"
            values={{ count: stashIDData?.findStudios.studios.length }}
          />
          :<b className="ml-2">{studio?.name}</b>
        </div>
        <span className="ml-auto">
          <SuccessIcon className="mr-2" />
          Matched:
        </span>
        <b className="col-3 text-right">
          {stashIDData.findStudios.studios[0].name}
        </b>
      </div>
    );
  }

  return (
    <div className="row no-gutters align-items-center mt-2">
      <Modal
        show={modalVisible}
        accept={{
          text: intl.formatMessage({ id: "actions.save" }),
          onClick: handleStudioCreate,
        }}
        cancel={{ onClick: () => showModal(false), variant: "secondary" }}
      >
        <div className="row">
          <strong className="col-2">
            <FormattedMessage id="name" />:
          </strong>
          <span className="col-10">{studio?.name}</span>
        </div>
        <div className="row">
          <strong className="col-2">
            <FormattedMessage id="url" />:
          </strong>
          <span className="col-10">{studio?.url ?? ""}</span>
        </div>
        <div className="row">
          <strong className="col-2">Logo:</strong>
          <span className="col-10">
            <img src={studio?.image ?? ""} alt="" />
          </span>
        </div>
      </Modal>

      <div className="entity-name">
        <FormattedMessage id="studios" />:<b className="ml-2">{studio?.name}</b>
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
          onClick={() => handleStudioSkip()}
        >
          <FormattedMessage id="actions.skip" />
        </Button>
        <StudioSelect
          ids={selectedStudio ? [selectedStudio] : []}
          onSelect={handleStudioSelect}
          className={cx("studio-select", {
            "studio-select-active": selectedSource === "existing",
          })}
          isClearable={false}
        />
      </ButtonGroup>
    </div>
  );
};

export default StudioResult;
