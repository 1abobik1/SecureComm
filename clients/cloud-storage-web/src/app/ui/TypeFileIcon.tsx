import React from 'react';
import {
  HomeTwoTone,
  FileTextTwoTone,
  CameraTwoTone ,
  FileUnknownTwoTone,
  PlayCircleTwoTone
} from '@ant-design/icons';



type TypeFileIcon = 'text' | 'photo' | 'video' |'unknown'|'home'  ;

interface FileIconProps {
  type: string;
  size?: number;
  color?: string;
}





const TypeFileIcon: React.FC<FileIconProps> = ({ type, size = 24 }) => {
  const iconMap: Record<TypeFileIcon, React.ReactNode> = {
    home: <HomeTwoTone  color="###1890ff"/>,
    text: <FileTextTwoTone   color="###1890ff" />,
    photo: <CameraTwoTone  color="##1890ff" />,
    video:  <PlayCircleTwoTone color="##1890ff" />,
    unknown: <FileUnknownTwoTone color="##1890ff" />,
  };

  const iconType = (type.toLowerCase() as TypeFileIcon) in iconMap ? (type.toLowerCase() as TypeFileIcon) : 'unknown';
  const IconComponent = iconMap[iconType];

  return (
    <span style={{ fontSize: size }}>
      {IconComponent}
    </span>
  );
};

export default TypeFileIcon;
