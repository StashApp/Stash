import _ from "lodash";
import React, { useEffect, useRef, useState } from "react";
import { Button, Form, Modal } from "react-bootstrap";
import { FilterSelect, DurationInput } from "src/components/Shared";
import { CriterionModifier } from "src/core/generated-graphql";
import {
  DurationCriterion,
  CriterionValue,
  Criterion,
  IHierarchicalLabeledIdCriterion,
} from "src/models/list-filter/criteria/criterion";
import {
  NoneCriterion,
  NoneCriterionOption,
} from "src/models/list-filter/criteria/none";
import { makeCriteria } from "src/models/list-filter/criteria/factory";
import { ListFilterOptions } from "src/models/list-filter/filter-options";
import { defineMessages, FormattedMessage, useIntl } from "react-intl";
import {
  criterionIsHierarchicalLabelValue,
  CriterionType,
} from "src/models/list-filter/types";

interface IAddFilterProps {
  onAddCriterion: (
    criterion: Criterion<CriterionValue>,
    oldId?: string
  ) => void;
  onCancel: () => void;
  filterOptions: ListFilterOptions;
  editingCriterion?: Criterion<CriterionValue>;
}

export const AddFilterDialog: React.FC<IAddFilterProps> = ({
  onAddCriterion,
  onCancel,
  filterOptions,
  editingCriterion,
}) => {
  const defaultValue = useRef<string | number | undefined>();

  const [criterion, setCriterion] = useState<Criterion<CriterionValue>>(
    new NoneCriterion()
  );
  const { options, modifierOptions } = criterion.criterionOption;

  const valueStage = useRef<CriterionValue>(criterion.value);

  const intl = useIntl();

  const messages = defineMessages({
    studio_depth: {
      id: "studio_depth",
      defaultMessage: "Levels (empty for all)",
    },
  });

  // Configure if we are editing an existing criterion
  useEffect(() => {
    if (!editingCriterion) {
      setCriterion(makeCriteria());
    } else {
      setCriterion(editingCriterion);
    }
  }, [editingCriterion]);

  function onChangedCriteriaType(event: React.ChangeEvent<HTMLSelectElement>) {
    const newCriterionType = event.target.value as CriterionType;
    const newCriterion = makeCriteria(newCriterionType);
    setCriterion(newCriterion);
  }

  function onChangedModifierSelect(
    event: React.ChangeEvent<HTMLSelectElement>
  ) {
    const newCriterion = _.cloneDeep(criterion);
    newCriterion.modifier = event.target.value as CriterionModifier;
    setCriterion(newCriterion);
  }

  function onChangedSingleSelect(event: React.ChangeEvent<HTMLSelectElement>) {
    const newCriterion = _.cloneDeep(criterion);
    newCriterion.value = event.target.value;
    setCriterion(newCriterion);
  }

  function onChangedInput(event: React.ChangeEvent<HTMLInputElement>) {
    valueStage.current = event.target.value;
  }

  function onChangedDuration(valueAsNumber: number) {
    valueStage.current = valueAsNumber;
    onBlurInput();
  }

  function onBlurInput() {
    const newCriterion = _.cloneDeep(criterion);
    newCriterion.value = valueStage.current;
    setCriterion(newCriterion);
  }

  function onAddFilter() {
    if (!Array.isArray(criterion.value) && defaultValue.current !== undefined) {
      const value = defaultValue.current;
      if (
        options &&
        (value === undefined || value === "" || typeof value === "number")
      ) {
        criterion.value = options[0].toString();
      } else if (typeof value === "number" && value === undefined) {
        criterion.value = 0;
      } else if (value === undefined) {
        criterion.value = "";
      }
    }
    const oldId = editingCriterion ? editingCriterion.getId() : undefined;
    onAddCriterion(criterion, oldId);
  }

  const maybeRenderFilterPopoverContents = () => {
    if (criterion.criterionOption.type === "none") {
      return;
    }

    function renderModifier() {
      if (modifierOptions.length === 0) {
        return;
      }
      return (
        <Form.Control
          as="select"
          onChange={onChangedModifierSelect}
          value={criterion.modifier}
          className="btn-secondary"
        >
          {modifierOptions.map((c) => (
            <option key={c.value} value={c.value}>
              {c.label ? intl.formatMessage({ id: c.label }) : ""}
            </option>
          ))}
        </Form.Control>
      );
    }

    function renderSelect() {
      // Hide the value select if the modifier is "IsNull" or "NotNull"
      if (
        criterion.modifier === CriterionModifier.IsNull ||
        criterion.modifier === CriterionModifier.NotNull
      ) {
        return;
      }

      if (Array.isArray(criterion.value)) {
        if (
          criterion.criterionOption.type !== "performers" &&
          criterion.criterionOption.type !== "studios" &&
          criterion.criterionOption.type !== "parent_studios" &&
          criterion.criterionOption.type !== "movies"
        )
          return;

        return (
          <FilterSelect
            type={criterion.criterionOption.type}
            isMulti
            onSelect={(items) => {
              const newCriterion = _.cloneDeep(criterion);
              newCriterion.value = items.map((i) => ({
                id: i.id,
                label: i.name!,
              }));
              setCriterion(newCriterion);
            }}
            ids={criterion.value.map((labeled) => labeled.id)}
          />
        );
      }
      if (criterion instanceof IHierarchicalLabeledIdCriterion) {
        if (
          criterion.criterionOption.type !== "studios" &&
          criterion.criterionOption.type !== "tags" &&
          criterion.criterionOption.type !== "sceneTags" &&
          criterion.criterionOption.type !== "performerTags"
        )
          return;

        return (
          <FilterSelect
            type={criterion.criterionOption.type}
            isMulti
            onSelect={(items) => {
              const newCriterion = _.cloneDeep(criterion);
              newCriterion.value.items = items.map((i) => ({
                id: i.id,
                label: i.name!,
              }));
              setCriterion(newCriterion);
            }}
            ids={criterion.value.items.map((labeled) => labeled.id)}
          />
        );
      }
      if (options && !criterionIsHierarchicalLabelValue(criterion.value)) {
        defaultValue.current = criterion.value;
        return (
          <Form.Control
            as="select"
            onChange={onChangedSingleSelect}
            value={criterion.value.toString()}
            className="btn-secondary"
          >
            {options.map((c) => (
              <option key={c.toString()} value={c.toString()}>
                {c}
              </option>
            ))}
          </Form.Control>
        );
      }
      if (criterion instanceof DurationCriterion) {
        // render duration control
        return (
          <DurationInput
            numericValue={criterion.value ? criterion.value : 0}
            onValueChange={onChangedDuration}
          />
        );
      }
      return (
        <Form.Control
          className="btn-secondary"
          type={criterion.criterionOption.inputType}
          onChange={onChangedInput}
          onBlur={onBlurInput}
          defaultValue={criterion.value ? criterion.value.toString() : ""}
        />
      );
    }
    function renderAdditional() {
      if (criterion instanceof IHierarchicalLabeledIdCriterion) {
        return (
          <>
            <Form.Group>
              <Form.Check
                checked={criterion.value.depth !== 0}
                label={intl.formatMessage({ id: "include_children" })}
                onChange={() => {
                  const newCriterion = _.cloneDeep(criterion);
                  newCriterion.value.depth =
                    newCriterion.value.depth !== 0 ? 0 : -1;
                  setCriterion(newCriterion);
                }}
              />
            </Form.Group>
            {criterion.value.depth !== 0 && (
              <Form.Group>
                <Form.Control
                  className="btn-secondary"
                  type="number"
                  placeholder={intl.formatMessage(messages.studio_depth)}
                  onChange={(e) => {
                    const newCriterion = _.cloneDeep(criterion);
                    newCriterion.value.depth = e.target.value
                      ? parseInt(e.target.value, 10)
                      : -1;
                    setCriterion(newCriterion);
                  }}
                  defaultValue={
                    criterion.value && criterion.value.depth !== -1
                      ? criterion.value.depth
                      : ""
                  }
                  min="1"
                />
              </Form.Group>
            )}
          </>
        );
      }
    }
    return (
      <>
        <Form.Group>{renderModifier()}</Form.Group>
        <Form.Group>{renderSelect()}</Form.Group>
        {renderAdditional()}
      </>
    );
  };

  function maybeRenderFilterCriterion() {
    if (!editingCriterion) {
      return;
    }

    return (
      <Form.Group>
        <strong>
          {intl.formatMessage({
            id: editingCriterion.criterionOption.messageID,
          })}
        </strong>
      </Form.Group>
    );
  }

  function maybeRenderFilterSelect() {
    if (editingCriterion) {
      return;
    }

    const thisOptions = [NoneCriterionOption]
      .concat(filterOptions.criterionOptions)
      .map((c) => {
        return {
          value: c.type,
          text: intl.formatMessage({ id: c.messageID }),
        };
      })
      .sort((a, b) => {
        if (a.value === "none") return -1;
        if (b.value === "none") return 1;
        return a.text.localeCompare(b.text);
      });

    return (
      <Form.Group controlId="filter">
        <Form.Label>
          <FormattedMessage id="search_filter.name" />
        </Form.Label>
        <Form.Control
          as="select"
          onChange={onChangedCriteriaType}
          value={criterion.criterionOption.type}
          className="btn-secondary"
        >
          {thisOptions.map((c) => (
            <option key={c.value} value={c.value} disabled={c.value === "none"}>
              {c.text}
            </option>
          ))}
        </Form.Control>
      </Form.Group>
    );
  }

  const title = !editingCriterion
    ? intl.formatMessage({ id: "search_filter.add_filter" })
    : intl.formatMessage({ id: "search_filter.update_filter" });
  return (
    <>
      <Modal show onHide={() => onCancel()}>
        <Modal.Header>{title}</Modal.Header>
        <Modal.Body>
          <div className="dialog-content">
            {maybeRenderFilterSelect()}
            {maybeRenderFilterCriterion()}
            {maybeRenderFilterPopoverContents()}
          </div>
        </Modal.Body>
        <Modal.Footer>
          <Button
            onClick={onAddFilter}
            disabled={criterion.criterionOption.type === "none"}
          >
            {title}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};
