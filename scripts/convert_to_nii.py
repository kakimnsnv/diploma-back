#!/usr/bin/env python3
# scripts/convert_to_nii.py

import sys
import numpy as np
import nibabel as nib
from PIL import Image
import argparse

def image_to_nii(input_path, output_path):
    """
    Convert PNG/JPEG image to NII format.
    
    Args:
        input_path: Path to input image (PNG/JPEG)
        output_path: Path to output NII file
    """
    try:
        # Load image
        img = Image.open(input_path)
        
        # Convert to RGB if needed
        if img.mode != 'RGB':
            img = img.convert('RGB')
        
        # Convert to numpy array
        img_array = np.array(img)
        
        # For medical imaging, we typically need grayscale
        # Convert RGB to grayscale using standard formula
        if len(img_array.shape) == 3:
            grayscale = np.dot(img_array[...,:3], [0.299, 0.587, 0.114])
            img_array = grayscale.astype(np.float32)
        
        # Add a third dimension to make it 3D (required for NII)
        # This creates a single slice volume
        img_3d = np.expand_dims(img_array, axis=2)
        
        # Normalize to 0-1 range
        img_3d = img_3d / 255.0
        
        # Create affine transformation matrix (identity with 1mm spacing)
        affine = np.eye(4)
        
        # Create NIfTI image
        nii_img = nib.Nifti1Image(img_3d, affine)
        
        # Save NII file
        nib.save(nii_img, output_path)
        
        print(f"Successfully converted {input_path} to {output_path}")
        print(f"Output shape: {img_3d.shape}")
        return 0
        
    except Exception as e:
        print(f"Error converting image: {str(e)}", file=sys.stderr)
        return 1

def nii_to_image(input_path, output_path):
    """
    Convert NII file back to PNG image.
    
    Args:
        input_path: Path to input NII file
        output_path: Path to output image (PNG)
    """
    try:
        # Load NII file
        nii_img = nib.load(input_path)
        img_data = nii_img.get_fdata()
        
        # If 3D, take the middle slice or first slice
        if len(img_data.shape) == 3:
            if img_data.shape[2] > 1:
                slice_idx = img_data.shape[2] // 2
                img_2d = img_data[:, :, slice_idx]
            else:
                img_2d = img_data[:, :, 0]
        else:
            img_2d = img_data
        
        # Normalize to 0-255 range
        img_2d = ((img_2d - img_2d.min()) / (img_2d.max() - img_2d.min()) * 255).astype(np.uint8)
        
        # Create PIL image and save
        pil_img = Image.fromarray(img_2d)
        pil_img.save(output_path)
        
        print(f"Successfully converted {input_path} to {output_path}")
        return 0
        
    except Exception as e:
        print(f"Error converting NII: {str(e)}", file=sys.stderr)
        return 1

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Convert between image and NII formats')
    parser.add_argument('input', help='Input file path')
    parser.add_argument('output', help='Output file path')
    parser.add_argument('--reverse', action='store_true', help='Convert NII to image instead')
    
    args = parser.parse_args()
    
    if args.reverse:
        sys.exit(nii_to_image(args.input, args.output))
    else:
        sys.exit(image_to_nii(args.input, args.output))