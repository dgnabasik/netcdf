import Validate from "./components/Validate/Validate";
import Dropdown from "./components/Dropdown/Dropdown";
import Checkbox from "./components/Checkbox/Checkbox";
import Multiple from "./components/Multiple/Multiple";
import Uncontrolled from "./components/Uncontrolled/Uncontrolled";
import Controlled from "./components/Controlled/Controlled";
import ReactHookForm from "./components/ReackHookForm/Example";
import Header from './components/Header/Header';

const App = () => {
  return (
    <div>
      <Header />
      {/* <Controlled /> */}
      {/* <Dropdown /> */}
      {/* <Checkbox /> */}
      {/* <Multiple /> */}
      {/* <Validate /> */}
      {/* <Uncontrolled /> */}
      <ReactHookForm />
    </div>
  );
};

export default App;
