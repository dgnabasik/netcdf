// UserGroups.tsx  
// bind() is unnecessary when using arrow function for lexical binding.
import { useState } from "react";
import Checkbox from "./Checkbox";

export interface IUserGroupsProps {
  names: string[];
};

function UserGroups({UserGroupProps}) { 
  //const userGroupProps: Map<string, string> = UserGroupProps;
  //const userGroupKeys:string[] = [ ...UserGroupProps.keys() ];
  //const userGroupValues:string[] = [ ...UserGroupProps.values() ];

  const [isChecked, setIsChecked] = useState(false);

  const [checkedState, setCheckedState] = useState(
    new Array(UserGroupProps.length).fill(false)
  );

  const handleOnChange = (position:number) => {
    const updatedCheckedState = checkedState.map((item, index) =>
      index === position ? !item : item
    );
    setCheckedState(updatedCheckedState);
  };

  const handleChange = (event) => {
    setIsChecked(event.target.checked);
  };
//                  <label htmlFor={`custom-checkbox-${index}`}>{name}</label>
  return (
    <div className="usergroups">
      <h5>Available Components</h5>
      <ul className="usergroups-list">
        {UserGroupProps.map(({ name }, index) => {
          return (
            <li key={index}>
              <div className="usergroups-list-item">
                <div className="left-section">
                  <Checkbox cboxKey={name} cboxValue={name} />
                  <label>{name}</label>
                </div>
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );

/*  return (
    <div className="usergroups">
      <h5>Available Components</h5>
      <ul className="usergroups-list">
              <div className="usergroups-list-item">
                <div className="left-section">
                  <input id= "IotTimeSeries_Authentication"
                    className="checkbox__input"
                    type="checkbox"
                    name={"IotTimeSeries_Authentication"}
                    checked={true}
                    onChange={handleChange}
                  />
                  <label>IoT Timeseries - Authentication</label>
                </div>
              </div>
      </ul>
    </div>
  ); */
}
//       {isChecked && <div>Blue is selected!</div>}

export default UserGroups;
