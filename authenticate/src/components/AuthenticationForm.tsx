// AuthenticationForm.tsx   https://mailboxlayer.com/documentation
import React, { useState, useEffect, useRef, forwardRef } from "react";
import { useForm, SubmitHandler } from "react-hook-form";
import "./css/authenticationForm.css";
import validator from "validator";
import TextareaAutosize from '@mui/base/TextareaAutosize';
import { styled } from '@mui/system';
//import Box from "@mui/material";

// https://email-checker.net/check  https://www.verifyemailaddress.org/
// https://www.digitalocean.com/community/tutorials/how-to-set-up-ssh-keys-on-ubuntu-20-04
// TODO: email password to client.
// TODO: output rdf code; button to upload.
// TODO: INSERT into GraphDB.

// Integrate with ServicesUser struct in user_data.go
export interface IAuthenticationProps {
  options?: { mbox: string; maker: string; firstName: string; lastName: string; password: string; sha1: string; rdf: string | number }[];
  mbox: string;
  maker: string;
  firstName: string;
  lastName: string;
  password: string;
  sha1: string;
  rdf: string;
}; 

export const UserServiceGroups = new Map<string, string>([
  ["IotTimeSeries_Authentication", "IoT Timeseries - Authentication"],
  ["IotTimeSeries_InferencePipeline", "IoT Timeseries - Inference Pipeline"],
  ["IotTimeSeries_SensorStream", "IoT Timeseries - Sensor Stream"],
  ["IotTimeSeries_RDFgraph", "IoT Timeseries - RDF Graph"],
  ["IotTimeSeries_AnalysisEngine", "IoT Timeseries - Analysis Engine"],
  ["IotTimeSeries_ChocolateSimulation", "IoT Timeseries - Chocolate Simulation"]
]);

function AuthenticationForm() {

  function EmptyTextarea() {
    const blue = {
      100: '#DAECFF',
      200: '#b6daff',
      400: '#3399FF',
      500: '#007FFF',
      600: '#0072E5',
      900: '#003A75',
    };

    const grey = {
      50: '#f6f8fa',
      100: '#eaeef2',
      200: '#d0d7de',
      300: '#afb8c1',
      400: '#8c959f',
      500: '#6e7781',
      600: '#57606a',
      700: '#424a53',
      800: '#32383f',
      900: '#24292f',
    };

  const StyledTextarea = styled(TextareaAutosize)(
    ({ theme }) => `
    width: 320px;
    font-family: IBM Plex Sans, sans-serif;
    font-size: 0.875rem;
    font-weight: 400;
    line-height: 1.5;
    padding: 12px;
    border-radius: 12px 12px 0 12px;
    color: ${theme.palette.mode === 'dark' ? grey[300] : grey[900]};
    background: ${theme.palette.mode === 'dark' ? grey[900] : '#fff'};
    border: 1px solid ${theme.palette.mode === 'dark' ? grey[700] : grey[200]};
    box-shadow: 0px 2px 24px ${
      theme.palette.mode === 'dark' ? blue[900] : blue[100]
    };
  
    &:hover {
      border-color: ${blue[400]};
    }
  
    &:focus {
      border-color: ${blue[400]};
      box-shadow: 0 0 0 3px ${theme.palette.mode === 'dark' ? blue[600] : blue[200]};
    }
  
    // firefox
    &:focus-visible {
      outline: 0;
    }
  `,
  );

  return <StyledTextarea aria-label="empty textarea" placeholder="Empty" />;
}

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<IAuthenticationProps>({
    mode: "onBlur"
  });   
  
  const [validEmail, setValidEmail] = useState(false);
  const [errorMessage, setErrorMessage] = useState("")
  const [_firstName, setFirstName] = useState("")
  const [_lastName, setLastName] = useState("")
  const [_maker, setMaker] = useState("")
  const [_rdf, setRdf] = useState("")

  const inputRef = useRef<HTMLInputElement | null>(null);
  useEffect(() => {
    inputRef.current?.focus();
  }, []);
  
  const onSubmit: SubmitHandler<IAuthenticationProps> = (data)  => {
    validateEmail(data.mbox).then(validEmail => {  // async promise
      validEmail;
    }); 
    data.firstName = _firstName;
    data.lastName = _lastName;
    data.maker = _maker;
    if (!validEmail || !validateMaker(data.maker) || !validatePassword(data.password) || !validateSha1(data.sha1)) 
    {
      console.log("onSubmit error");
      return
    }
    data.rdf = formatFOAF(data, UserServiceGroups);
    setRdf(data.rdf);
    uploadRdf(data.rdf).then(validRdf => {  // async promise
      validRdf;
    }); 
    //<<< upload data.rdf to GraphDB.
    // display pretty RDF https://www.npmjs.com/package/@rdfjs-elements/formats-pretty/v/0.4.3
  };

  const uploadRdf = async (rdf:string) => {
    const response = await fetch(rdf);
    await response.json();
  }

  // async functions always return a promise.
  const validateEmail = async (value:string) => {
    if (!value.toLowerCase().includes("@emse.fr")) {
      setErrorMessage("only @emse.fr mailboxes accepted...")
      setValidEmail(false);  
      return;
    }
    const apiKey:string = "8edb0e203e0c2d1a9e87746f64de9a18";
    const apiUrl: string = "http://apilayer.net/api/check?access_key=" + apiKey + "&email=" + value + "&smtp=1&format=1";
    const response = await fetch(apiUrl);
    const mboxJson = await response.json(); //extract JSON from the http response
    const result:boolean = !mboxJson.hasOwnProperty("success");
    setValidEmail(result);
    if (result) {
      const names:string[] = mboxJson.user.split(".");
      setFirstName(names[0]);
      setLastName(names[1]);
      setMaker(names[0]+names[1])
    }
  }

  function validateMaker(value:string):boolean {  
    const result:boolean = value.length > 3;
    return result;
  }

  function validatePassword(value:string):boolean {  
    if (validator.isStrongPassword(value, {
        minLength: 8, minLowercase: 1,
        minUppercase: 1, minNumbers: 1, minSymbols: 1
    })) {
        setErrorMessage("Password is strong!")
        return true;
      } else {
        setErrorMessage("Password is weak...")
        return false;
      }
  }

  // Default RSA key length is 3072 bits.
  function validateSha1(value:string):boolean { 
    if (value.length < 3) {
      return true
    }
    const pieces:string[] = value.split(" ");
    const result:boolean = (pieces.length == 3 && pieces[0] != "ssh-rsa" && pieces[1].length >= 384);
    return result;
  }

  function formatFOAF(data: IAuthenticationProps, groupMap: Map<string, string>):string {
    var foaf:string = `<rdf:RDF
      xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
      xmlns:rdfs="http://www.w3.org/2000/01/rdf-schema#"
      xmlns:foaf="http://xmlns.com/foaf/0.1/"
      xmlns:dc="http://purl.org/dc/elements/1.1/"
      xmlns:bibo="http://purl.org/ontology/bibo/"
      xmlns:dcterms="http://purl.org/dc/terms/"
      xmlns:ical="http://www.w3.org/2002/12/cal/ical#"
      xmlns:geo="http://www.w3.org/2003/01/geo/wgs84_pos#" >
  <foaf:PersonalProfileDocument rdf:about="">
    <foaf:maker rdf:resource="#${data.maker}"/>
    <foaf:primaryTopic rdf:resource="#${data.maker}"/>
  </foaf:PersonalProfileDocument>
 <foaf:Person>
   <foaf:name>${data.firstName + " " + data.lastName}</foaf:name>
   <foaf:Organization>École des Mines de Saint-Étienne</foaf:Organization>
   <foaf:mbox>${data.mbox}</foaf:mbox>
   <foaf:sha1>${data.sha1}</foaf:sha1>
   `;

    foaf += `<!-- knows -->`;
    for (let [key, value] of groupMap) {
      foaf += `<foaf:knows rdf:resource="#` + key + `"/>`;
    } 
    foaf += `</foaf:Person>`;
    foaf += `<!-- groups -->`;
    for (let [key, value] of groupMap) {
      foaf += `<foaf:Group rdf:ID="` + key + `">`;
      foaf += `<foaf:name>` + value + `</foaf:name>`;
      foaf += `<foaf:member rdf:resource="#${data.maker}"/>`;
      foaf += `</foaf:Group>`;
    } 
    foaf += `</rdf:RDF>`;
    foaf = foaf.replace(/\s/g," ");
    return foaf;
  }

 // handleSubmit() validates inputs before invoking onSubmit() 
  return (
    <form onSubmit={handleSubmit(onSubmit)} className="hook">
      <p>Enter your emse.fr email into this Friend-of-a-Friend form for access to MINES ontology resources.</p>
      <label className="hook__text">emse.fr Email</label>
      <input
        type="email"
        className="hook__input"
        {...register("mbox", { required: true, pattern: /^\S+@\S+$/i })}
      />
      {errors.mbox && (
        <p className="hook__error">emse.fr email is required and must be valid</p>
      )}

      <label className="hook__text">Password</label>
      <input
        type="password"
        className="hook__input"
        {...register("password", { required: true })}        
      />
      {errors.password && <p className="hook__error">Password is required</p>}

      <label className="hook__text">You will be granted Administrative access if you enter your complete (3 parts) public key that you generated on your computer using ssh-keygen.</label>
      <label className="hook__text">Open that file (~/.ssh/[mynewkey].pub) This public key will be added to the authentication server.</label>
      <input
        type="text"
        className="hook__input"
        {...register("sha1", { required: false })}
      />

      <button className="hook__button" type="submit">
        Validate
      </button>

      <p>{_rdf}</p>

    </form>
  );
}
export default AuthenticationForm;
/*
      <StyledTextarea
        aria-label="maximum height"
        defaultValue="Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt
          ut labore et dolore magna aliqua."
      />

    <label className="hook__text">Unique rdf:resource Identifier such as FirstNameLastName (no dot, no space)</label>
      <input
        type="text"
        placeholder={_firstName+_lastName}
        className="hook__input"
        {...register("maker", { required: true })}
      />
      {errors.maker && (
        <p className="hook__error">Identifier is required and must be valid</p>
      )}

      <label className="hook__text">Your First & Last Names</label>
      <input
        type="text"
        placeholder={_firstName+" "+_lastName}
        className="hook__input"
        {...register("name", { required: true })}
      />
      {errors.maker && (
        <p className="hook__error">Please enter your first & last names</p>
      )}
*/
