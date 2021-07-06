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
  NumberCriterion,
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
  criterionIsNumberValue,
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

  useEffect(() => {
    valueStage.current = criterion.value;
  }, [criterion]);

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

  function onNumberChanged(
    event: React.ChangeEvent<HTMLInputElement>,
    property: "exact" | "lower" | "upper"
  ) {
    const value = parseInt(event.target.value, 10);

    if (!criterionIsNumberValue(valueStage.current)) {
      valueStage.current = { exact: 0, upper: 0, lower: 0 };
    }

    valueStage.current[property] = value;
  }

  function onChangedDuration(
    valueAsNumber: number,
    property: "exact" | "lower" | "upper"
  ) {
    if (!criterionIsNumberValue(valueStage.current)) {
      valueStage.current = { exact: 0, upper: 0, lower: 0 };
    }

    valueStage.current[property] = valueAsNumber;
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
      } else if (
        (criterion instanceof NumberCriterion ||
          criterion instanceof DurationCriterion) &&
        value === undefined
      ) {
        criterion.value = { exact: 0, lower: 0, upper: 0 };
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
          criterion.criterionOption.type !== "tags" &&
          criterion.criterionOption.type !== "sceneTags" &&
          criterion.criterionOption.type !== "performerTags" &&
          criterion.criterionOption.type !== "movies"
        )
          return;

        return (
          <Form.Group>
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
          </Form.Group>
        );
      }
      if (criterion instanceof IHierarchicalLabeledIdCriterion) {
        if (criterion.criterionOption.type !== "studios") return;

        return (
          <Form.Group>
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
          </Form.Group>
        );
      }
      if (
        options &&
        !criterionIsHierarchicalLabelValue(criterion.value) &&
        !criterionIsNumberValue(criterion.value)
      ) {
        defaultValue.current = criterion.value;
        return (
          <Form.Group>
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
          </Form.Group>
        );
      }
      if (criterion instanceof DurationCriterion) {
        if (
          criterion.modifier === CriterionModifier.Equals ||
          criterion.modifier === CriterionModifier.NotEquals
        ) {
          return (
            <Form.Group>
              <DurationInput
                numericValue={
                  criterion.value?.exact ? criterion.value.exact : 0
                }
                onValueChange={(v: number) => onChangedDuration(v, "exact")}
              />
            </Form.Group>
          );
        }

        let lowerControl: JSX.Element | null = null;
        if (
          criterion.modifier === CriterionModifier.GreaterThan ||
          criterion.modifier === CriterionModifier.Between ||
          criterion.modifier === CriterionModifier.NotBetween
        ) {
          lowerControl = (
            <Form.Group>
              <DurationInput
                numericValue={
                  criterion.value?.lower ? criterion.value.lower : 0
                }
                onValueChange={(v: number) => onChangedDuration(v, "lower")}
              />
            </Form.Group>
          );
        }

        let upperControl: JSX.Element | null = null;
        if (
          criterion.modifier === CriterionModifier.LessThan ||
          criterion.modifier === CriterionModifier.Between ||
          criterion.modifier === CriterionModifier.NotBetween
        ) {
          upperControl = (
            <Form.Group>
              <DurationInput
                numericValue={
                  criterion.value?.upper ? criterion.value.upper : 0
                }
                onValueChange={(v: number) => onChangedDuration(v, "upper")}
              />
            </Form.Group>
          );
        }

        return (
          <>
            {lowerControl}
            {upperControl}
          </>
        );
      }
      if (criterion instanceof NumberCriterion) {
        if (
          criterion.modifier === CriterionModifier.Equals ||
          criterion.modifier === CriterionModifier.NotEquals
        ) {
          return (
            <Form.Group>
              <Form.Control
                className="btn-secondary"
                type="number"
                onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                  onNumberChanged(e, "exact")
                }
                onBlur={onBlurInput}
                defaultValue={criterion.value?.exact ?? ""}
              />
            </Form.Group>
          );
        }

        let lowerControl: JSX.Element | null = null;
        if (
          criterion.modifier === CriterionModifier.GreaterThan ||
          criterion.modifier === CriterionModifier.Between ||
          criterion.modifier === CriterionModifier.NotBetween
        ) {
          lowerControl = (
            <Form.Group>
              <Form.Control
                className="btn-secondary"
                type="number"
                onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                  onNumberChanged(e, "lower")
                }
                onBlur={onBlurInput}
                defaultValue={criterion.value?.lower ?? ""}
              />
            </Form.Group>
          );
        }

        let upperControl: JSX.Element | null = null;
        if (
          criterion.modifier === CriterionModifier.LessThan ||
          criterion.modifier === CriterionModifier.Between ||
          criterion.modifier === CriterionModifier.NotBetween
        ) {
          upperControl = (
            <Form.Group>
              <Form.Control
                className="btn-secondary"
                type="number"
                onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                  onNumberChanged(e, "upper")
                }
                onBlur={onBlurInput}
                defaultValue={criterion.value?.upper ?? ""}
              />
            </Form.Group>
          );
        }

        return (
          <>
            {lowerControl}
            {upperControl}
          </>
        );
      }
      return (
        <Form.Group>
          <Form.Control
            className="btn-secondary"
            type={criterion.criterionOption.inputType}
            onChange={onChangedInput}
            onBlur={onBlurInput}
            defaultValue={criterion.value ? criterion.value.toString() : ""}
          />
        </Form.Group>
      );
    }
    function renderAdditional() {
      if (criterion instanceof IHierarchicalLabeledIdCriterion) {
        return (
          <>
            <Form.Group>
              <Form.Check
                checked={criterion.value.depth !== 0}
                label={intl.formatMessage({ id: "include_child_studios" })}
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
        {renderSelect()}
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
